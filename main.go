package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

func main() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonhandler.Default)
	}
	addr := ":" + os.Getenv("PORT")
	app := mux.NewRouter()
	app.HandleFunc("/subscribe", handlePost).Methods("POST")
	app.HandleFunc("/", handleIndex).Methods("GET")
	var options []csrf.Option
	// If developing locally
	if os.Getenv("UP_STAGE") == "" {
		// https://godoc.org/github.com/gorilla/csrf#Secure
		log.Warn("CSRF insecure")
		options = append(options, csrf.Secure(false))
	}
	if err := http.ListenAndServe(addr,
		csrf.Protect([]byte("dabase"), options...)(app)); err != nil {
		log.WithError(err).Fatal("error listening")
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	t.ExecuteTemplate(w, "index.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"Stage":          os.Getenv("UP_STAGE"),
		"Year":           time.Now().Format("2006"),
	})
}

func handlePost(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.WithError(err).Error("parsing form")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for key, values := range r.PostForm { // range over map
		for _, value := range values { // range over []string
			log.Infof("Key: %v Value: %v", key, value)
		}
	}

	log.Infof("Email address: %s", r.PostForm["email"][0])

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mine"))
	if err != nil {
		log.WithError(err).Error("loading config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfg.Region = endpoints.ApEast1RegionID

	svc := sns.New(cfg)
	req := svc.SubscribeRequest(&sns.SubscribeInput{
		Endpoint: aws.String(r.PostForm["email"][0]),
		Protocol: aws.String("email"),
		TopicArn: aws.String("arn:aws:sns:ap-southeast-1:407461997746:dabase"),
	})
	_, err = req.Send(context.TODO())
	if err != nil {
		log.WithError(err).Error("unable to subscribe")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	t.ExecuteTemplate(w, "thankyou.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"Stage":          os.Getenv("UP_STAGE"),
		"Year":           time.Now().Format("2006"),
	})
}
