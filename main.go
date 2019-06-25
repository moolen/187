package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	flag "github.com/namsral/flag"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config contains the server (the webhook) cert and key.
type Config struct {
	CertFile    string
	KeyFile     string
	ListenAddr  string
	LogLevel    string
	GracePeriod time.Duration
}

var (
	config Config
	client *kubernetes.Clientset
)

func main() {
	var err error
	flag.StringVar(&config.LogLevel, "log-level", "info", "log level")
	flag.StringVar(&config.ListenAddr, "listen", ":8000", "port to listen on")
	flag.DurationVar(&config.GracePeriod, "grace-period", time.Minute*15, "time until the pod gets killed")
	flag.StringVar(&config.CertFile, "tls-cert", config.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	flag.StringVar(&config.KeyFile, "tls-key", config.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	flag.Parse()

	level, err := log.ParseLevel(config.LogLevel)
	if err == nil {
		log.SetLevel(level)
	}

	client, err = makeClient()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", servePods(client, config.GracePeriod))
	server := &http.Server{
		Addr:      config.ListenAddr,
		TLSConfig: configTLS(config),
	}
	log.Info("starting server")
	server.ListenAndServeTLS("", "")
}

func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

type admitFunc func(v1beta1.AdmissionReview, *kubernetes.Clientset, time.Duration) *v1beta1.AdmissionResponse

func serve(w http.ResponseWriter, r *http.Request, admit admitFunc, client *kubernetes.Clientset, gracePeriod time.Duration) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	var reviewResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Error(err)
		reviewResponse = toAdmissionResponse(err)
	} else {
		reviewResponse = admit(ar, client, gracePeriod)
	}
	log.Infof(fmt.Sprintf("sending response: %v", reviewResponse))

	response := v1beta1.AdmissionReview{}
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = ar.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	ar.Request.Object = runtime.RawExtension{}
	ar.Request.OldObject = runtime.RawExtension{}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		log.Error(err)
	}
}

func servePods(client *kubernetes.Clientset, gracePeriod time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, admitPods, client, gracePeriod)
	}
}

func makeClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	if clientSet == nil {
		return nil, fmt.Errorf("kubernetes client set cannot be nil")
	}
	return clientSet, nil
}
