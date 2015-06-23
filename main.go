package main

import (
	"flag"
	goetcd "github.com/coreos/go-etcd/etcd"
	godocker "github.com/fsouza/go-dockerclient"
	"log"
	"net/http"
	_ "net/http/pprof" // Adds http://*/debug/pprof/ to default mux.
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	endpoint        = flag.String("docker", "unix:///var/run/docker.sock", "Docker endpoint to connect to.")
	etcdNodes       = flag.String("etcd_nodes", "http://127.0.0.1:4001", "Comma-seperated list of etcd nodes to connect to.")
	etcdPrefix      = flag.String("etcd_prefix", "/deployer", "Path prefix for etcd nodes.")
	etcdDialTimeout = flag.Duration("etcd_dial_timeout", 5*time.Second, "How long to wait to connect to etcd nodes.")
	etcdRetryDelay  = flag.Duration("etcd_retry_delay", time.Second, "How long to between request retries.")
	dockerCfg       = flag.String("dockercfg", os.Getenv("HOME")+"/.dockercfg", "Path to .dockercfg authentication information.")
	listenAddr      = flag.String("listen", ":4500", "[IP]:port to listen for incoming connections.")
	maxThreads      = flag.Int("max_threads", runtime.NumCPU(), "Maximum number of running threads.")
	killTimeout     = flag.Int("kill_timeout", 10, "Container stop timeout, before hard kill (in seconds).")
	repullPeriod    = flag.Duration("repull_period", 24*time.Hour, "How frequently to re-pull all images, without any notification.")
	registry        = flag.String("registry", "https://index.docker.io/v1/", "URL of docker registry.")
	webhookPath     = flag.String("webhook_path", "/api/dockerhub/webhook", "Path to webhook from Docker Hub.")
)

func main() {
	flag.Parse()

	runtime.GOMAXPROCS(*maxThreads)

	docker, err := godocker.NewClient(*endpoint)
	if err != nil {
		log.Fatalln("Couldn't docker.NewClient: ", err)
	}

	etcd := goetcd.NewClient(strings.Split(*etcdNodes, ","))
	etcd.SetDialTimeout(*etcdDialTimeout)
	etcd.SetConsistency(goetcd.STRONG_CONSISTENCY)

	// Add a delay for each retry attempt
	etcd.CheckRetry = func(cluster *goetcd.Cluster, numReqs int, lastResp http.Response, err error) error {
		time.Sleep(*etcdRetryDelay)
		return goetcd.DefaultCheckRetry(cluster, numReqs, lastResp, err)
	}

	auth := godocker.AuthConfiguration{}
	if *dockerCfg != "" {
		auth, err = AuthFromDockerCfg(*dockerCfg, *registry)
		if err != nil {
			log.Fatalln("AuthFromDockerCfg: ", err)
		}
	}

	deployer := NewDeployer(docker, *registry, auth, etcd, *etcdPrefix, uint(*killTimeout), *repullPeriod)
	deployer.RegisterDockerHubWebhook(*webhookPath)

	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
