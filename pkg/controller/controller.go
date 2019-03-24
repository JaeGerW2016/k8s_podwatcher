package controller

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"k8s_podwatcher/pkg/handlers"
)

type Controller struct {
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
	handler   handlers.Handler
}

func NewController(clientset kubernetes.Interface, handler handlers.Handler, namespace string) *Controller {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (i watch.Interface, e error) {
				return clientset.CoreV1().Pods(namespace).Watch(options)
			},
			ListFunc: func(options metav1.ListOptions) (object runtime.Object, e error) {
				return clientset.CoreV1().Pods(namespace).List(options)
			},
		},
		&apiv1.Pod{},
		0,
		cache.Indexers{},
	)
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(oldObj)
			if err != nil {
				klog.Errorf("Unknown object:%v", oldObj)
			}
			queue.Add(key)

		},
	})
	return &Controller{
		clientset: clientset,
		handler:   handler,
		informer:  informer,
		queue:     queue,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	klog.Info("Starting podwatcher controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		c.queue.ShutDown()
		utilruntime.HandleError(fmt.Errorf("timeout for waiting for cache sync"))
		return
	}
	klog.Info("PodWatcher controller synced and ready")
	go func() {
		<-stopCh
		klog.Info("PodWatcher controller is shutting down")
		c.queue.ShutDown()
	}()
	c.runWorker()

}

func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}

}

func (c *Controller) processNextItem() bool {
	klog.Info("Controller.processNextItem: start")
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	keyRaw := key.(string)
	err := c.processItem(keyRaw)
	if err == nil {
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < 3 {
		klog.Errorf("Error processing %s (will retry): %v", key, err)
	} else {
		klog.Errorf("Error processing %s (giving up): %v", key, err)
		c.queue.Forget(key)
		utilruntime.HandleError(err)
	}
	return true
}

func (c *Controller) processItem(key string) error {
	item, _, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("error fetching item with key %s from store: %v", key, err)
	}
	if item == nil {
		return nil
	}
	pod := item.(*apiv1.Pod)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			var tailLines int64 = 20
			rawLog, err := c.clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &apiv1.PodLogOptions{
				Container: cs.Name,
				TailLines: &tailLines,
			}).DoRaw()
			if err != nil {
				return fmt.Errorf("get log failed:%v", err)
			}
			klog.Errorf("Container %s in pod %s crashed", cs.Name, key)
			return c.handler.Handle(&handlers.Event{
				Namespace:     pod.Namespace,
				Name:          pod.Name,
				ContainerName: cs.Name,
				Reason:        cs.State.Waiting.Reason,
				Message:       cs.State.Waiting.Message,
				RawLogs:       string(rawLog),
			})
		}
	}
	return nil
}
