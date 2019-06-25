package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// kill pod after grace period
func admitPods(ar v1beta1.AdmissionReview, client *kubernetes.Clientset, gracePeriod time.Duration) *v1beta1.AdmissionResponse {
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Error(err)
		return toAdmissionResponse(err)
	}
	period := int64(gracePeriod / time.Second)
	log.WithFields(log.Fields{
		"pod":                ar.Request.Name,
		"namespace":          ar.Request.Namespace,
		"gracePeriodSeconds": period,
	}).Infof("deleting pod")

	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &period,
	}
	err := client.CoreV1().Pods(ar.Request.Namespace).Delete(ar.Request.Name, deleteOptions)
	if err != nil {
		log.WithFields(log.Fields{
			"pod":       ar.Request.Name,
			"namespace": ar.Request.Namespace,
		}).WithError(err).Warn("unable to delete pod", err)
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}
