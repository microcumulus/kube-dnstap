package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const gb = 1024 * 1024 * 1024

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	l, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		log.Fatal(err)
	}

	_ = k8sMap(ctx)

	i := dnstap.NewFrameStreamSockInput(l)
	bch := make(chan []byte)
	go i.ReadInto(bch)

	for bs := range bch {
		r, err := dnstap.NewReader(bytes.NewReader(bs), nil)
		if err != nil {
			logrus.WithError(err).Error("failed to create dnstap reader")
			continue
		}
		d := dnstap.NewDecoder(r, gb)
		f := &dnstap.Dnstap{}
		err = d.Decode(f)
		if err != nil {
			logrus.WithError(err).Error("failed to decode dnstap message")
			continue
		}
		log.Println(f.String())
	}

}

func k8sMap(ctx context.Context) sync.Map {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "home")
	config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Pods().Informer()

	var m sync.Map

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			m.Store(pod.Status.PodIP, &pod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod := newObj.(*corev1.Pod)
			m.Store(newPod.Status.PodIP, &newPod)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			m.Delete(pod.Status.PodIP)
		},
	})

	go informer.Run(ctx.Done())
	return m
}
