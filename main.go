package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

var views = template.Must(template.New("").ParseGlob("templates/*.html"))

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
	if os.Getenv("UP_STAGE") == "" {
		log.Warn("CSRF insecure")
		options = append(options, csrf.Secure(false)) // https://godoc.org/github.com/gorilla/csrf#Secure
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
	views.ExecuteTemplate(w, "index.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
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

	subscriberEmail := r.PostForm["email"][0]
	ctx := log.WithField("email", subscriberEmail)

	sess := session.New()
	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{Filename: "", Profile: "mine"},
			&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(sess)},
		})
	cfg := &aws.Config{
		Region:                        aws.String("ap-southeast-1"),
		Credentials:                   creds,
		CredentialsChainVerboseErrors: aws.Bool(true),
	}
	sess, err = session.NewSession(cfg)
	if err != nil {
		ctx.WithError(err).Error("unable to start session")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	svc := sns.New(sess)
	_, err = svc.Subscribe(&sns.SubscribeInput{
		Endpoint: aws.String(subscriberEmail),
		Protocol: aws.String("email"),
		TopicArn: aws.String("arn:aws:sns:ap-southeast-1:407461997746:dabase"),
	})
	if err != nil {
		ctx.WithError(err).Error("unable to subscribe")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.ExecuteTemplate(w, "thankyou.html", nil)
	ctx.Info("subscribed")
}
