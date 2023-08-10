package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/docker/distribution/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	l, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		log.Fatal(err)
	}

	m := k8sMap(ctx)

	i := dnstap.NewFrameStreamSockInput(l)
	for i.Read() {
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

	go informer.Run(ctx.Done)
	return m
}
