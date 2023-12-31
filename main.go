package main

import (
	"net"
	"net/http"
	"reflect"
	"strings"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/gopuff/morecontext"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
)

var (
	query = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dnstap_dns_queries_total",
		Help: "The total number of dns queries",
	}, []string{"name", "type", "ns", "pod"})
)

func main() {
	cfg := setupConfig()
	ctx := morecontext.ForSignals()

	if !cfg.GetBool("metrics.disabled") {
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(cfg.GetString("metrics.addr"), nil)
	}

	l, err := net.Listen("tcp", cfg.GetString("listen.addr"))
	if err != nil {
		lg.WithError(err).Fatal("failed to listen")
	}
	lg.Info("listening")

	m := k8sMap(ctx)

	i := dnstap.NewFrameStreamSockInput(l)
	bch := make(chan []byte)
	go i.ReadInto(bch)

	ignores := cfg.GetStringSlice("suffixes.ignore")
	only := cfg.GetStringSlice("suffixes.only")

	for {
		select {
		case <-ctx.Done():
			return
		case bs := <-bch:
			var f dnstap.Dnstap
			err := proto.Unmarshal(bs, &f)
			if err != nil {
				lg.WithError(err).Error("failed to decode dnstap message")
				continue
			}
			addr := net.IP(f.Message.QueryAddress)

			v, ok := m.Load(addr.String())
			if !ok {
				lg.WithFields(logrus.Fields{
					"addr": addr,
				}).Error("no pod found")
				continue
			}
			pod, ok := v.(*corev1.Pod)
			if !ok {
				lg.WithField("addr", addr).WithField("type", reflect.TypeOf(v)).Error("cached value is not a pod")
				continue
			}
			lg := lg.WithFields(logrus.Fields{
				"pod": pod.Name,
				"ns":  pod.Namespace,
				// "addr": addr,
				// "tap":  f,
			})
			lg.Debug("dnstap message received")

			msgBs := f.Message.GetQueryMessage()
			if len(msgBs) == 0 {
				msgBs = f.Message.GetResponseMessage()
			}
			if len(msgBs) == 0 {
				lg.Error("no dns message found")
				continue
			}

			if *f.Message.Type != dnstap.Message_CLIENT_QUERY {
				lg.WithField("type", f.Message.Type.String()).Debug("not a query")
				continue
			}

			var msg dns.Msg
			err = msg.Unpack(msgBs)
			if err != nil {
				lg.WithError(err).Error("failed to decode dns message")
				continue
			}

		q:
			for _, n := range msg.Question {
				if len(only) > 0 {
					for _, suff := range only {
						if !strings.HasSuffix(n.Name, suff) {
							continue q
						}
					}
				}
				for _, suff := range ignores {
					if strings.HasSuffix(n.Name, suff) {
						continue q
					}
				}
				query.WithLabelValues(n.Name, dns.TypeToString[n.Qtype], pod.Namespace, pod.Name).Inc()

				if cfg.GetBool("noLog") {
					continue
				}
				lg.WithField("name", n.Name).Info()
			}
		}
	}
}
