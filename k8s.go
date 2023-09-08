package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func k8sMap(ctx context.Context) sync.Map {
	var cfg *rest.Config
	var err error
	if isK8s {
		// load incluster config
		cfg, err = rest.InClusterConfig()
	} else {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		lg.WithError(err).Fatal("failed to build config")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		lg.WithError(err).Fatal("failed to create clientset")
	}
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Pods().Informer()

	var m sync.Map

	list, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		lg.WithError(err).Fatal("failed to list pods")
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
		lg.WithError(err).Fatal("failed to add event handler")
	}

	go informer.Run(ctx.Done())
	return m
}
