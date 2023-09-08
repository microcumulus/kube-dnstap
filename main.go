package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/gopuff/morecontext"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var isK8s = os.Getenv("KUBERNETES_SERVICE_HOST") != ""

func main() {
	ctx := morecontext.ForSignals()

	if isK8s {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	l, err := net.Listen("tcp", "0.0.0.0:12345")
	if err != nil {
		log.Fatal(err)
	}

	m := k8sMap(ctx)

	i := dnstap.NewFrameStreamSockInput(l)
	bch := make(chan []byte)
	go i.ReadInto(bch)

	for {
		select {
		case <-ctx.Done():
			return
		case bs := <-bch:
			var f dnstap.Dnstap
			err := proto.Unmarshal(bs, &f)
			if err != nil {
				logrus.WithError(err).Error("failed to decode dnstap message")
				continue
			}
			addr := net.IP(f.Message.QueryAddress)

			v, ok := m.Load(addr.String())
			if !ok {
				logrus.WithFields(logrus.Fields{
					"addr": addr,
				}).Error("no pod found")
				continue
			}
			pod, ok := v.(*corev1.Pod)
			if !ok {
				logrus.WithField("addr", addr).WithField("type", reflect.TypeOf(v)).Error("cached value is not a pod")
				continue
			}
			lg := logrus.WithFields(logrus.Fields{
				"pod": pod.Name,
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
			var msg dns.Msg
			err = msg.Unpack(msgBs)
			if err != nil {
				logrus.WithError(err).Error("failed to decode dns message")
				continue
			}

			lg.WithField("type", f.Message.Type.String()).WithField("msg", msg.Question[0].Name).Info()
		}
	}
}

func k8sMap(ctx context.Context) sync.Map {
	var cfg *rest.Config
	var err error
	if isK8s {
		// load incluster config
		cfg, err = rest.InClusterConfig()
	} else {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "home")
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		logrus.WithError(err).Fatal("failed to build config")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create clientset")
	}
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Pods().Informer()

	var m sync.Map

	list, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Fatal("failed to list pods")
	}
	for _, pod := range list.Items {
		pod := pod
		m.Store(pod.Status.PodIP, &pod)
	}

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			m.Store(pod.Status.PodIP, pod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod := newObj.(*corev1.Pod)
			m.Store(newPod.Status.PodIP, newPod)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			m.Delete(pod.Status.PodIP)
		},
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to add event handler")
	}

	go informer.Run(ctx.Done())
	return m
}
