package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jamiehannaford/canary-operator/pkg/canary"
	"github.com/jamiehannaford/canary-operator/pkg/spec"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v1beta1extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

var (
	initRetryWaitTime = 30 * time.Second
)

type Config struct {
	Namespace string
	KubeCli   kubernetes.Interface
}

type Controller struct {
	config *Config

	canaries  map[string]*canary.Canary
	canaryRVs map[string]string
}

type Event struct {
	Type   watch.EventType
	Object *spec.Canary
}

func New(config *Config) Controller {
	return Controller{
		config:    config,
		canaries:  make(map[string]*canary.Canary),
		canaryRVs: make(map[string]string),
	}
}

func (c Controller) Run() error {
	var (
		watchVersion string
		err          error
	)

	// create canary TPR
	for {
		watchVersion, err = c.initTPR()
		if err == nil {
			break
		}
		fmt.Printf("initialization failed: %v\n", err)
		fmt.Printf("retry in %v...\n", initRetryWaitTime)
		time.Sleep(initRetryWaitTime)
	}

	fmt.Printf("starts running from watch version: %s\n", watchVersion)

	// create watch/error channels
	eventCh, errorCh := c.watchCanaries(watchVersion)

	// handle any canary resource related event
	go func() {
		for event := range eventCh {
			if err := c.handleCanaryEvent(event); err != nil {
				log.Fatal(err)
			}
		}
	}()

	// return any received errors immediately and break execution
	return <-errorCh
}

func (c Controller) createTPR() error {
	tpr := &v1beta1extensions.ThirdPartyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: spec.TPRName(),
		},
		Versions: []v1beta1extensions.APIVersion{
			{Name: spec.TPRVersion},
		},
		Description: spec.TPRDescription,
	}
	_, err := c.config.KubeCli.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil {
		return err
	}

	return waitEtcdTPRReady(c.config.KubeCli.CoreV1().RESTClient(), 3*time.Second, 30*time.Second, c.config.Namespace)
}

func (c Controller) initTPR() (string, error) {
	watchVersion := "0"
	err := c.createTPR()

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// TPR has been initialized before. We need to recover existing cluster.
			watchVersion, err = c.findAllCanaries()
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("fail to create TPR: %v", err)
		}
	}

	return watchVersion, nil
}

func (c *Controller) findAllCanaries() (string, error) {
	fmt.Println("finding existing canaries...")
	canaryList, err := c.getCanaryList(c.config.KubeCli.CoreV1().RESTClient(), c.config.Namespace)
	if err != nil {
		return "", err
	}

	for i := range canaryList.Items {
		can := canaryList.Items[i]

		nc := canary.New(&can)
		c.canaries[can.Metadata.Name] = nc
		c.canaryRVs[can.Metadata.Name] = can.Metadata.ResourceVersion
	}

	return canaryList.Metadata.ResourceVersion, nil
}

type RetryError struct {
	n int
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("still failing after %d retries", e.n)
}

func IsRetryFailure(err error) bool {
	_, ok := err.(*RetryError)
	return ok
}

type ConditionFunc func() (bool, error)

func retry(interval time.Duration, maxRetries int, f ConditionFunc) error {
	if maxRetries <= 0 {
		return fmt.Errorf("maxRetries (%d) should be > 0", maxRetries)
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()

	for i := 0; ; i++ {
		ok, err := f()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if i+1 == maxRetries {
			break
		}
		<-tick.C
	}
	return &RetryError{maxRetries}
}

func waitEtcdTPRReady(restcli rest.Interface, interval, timeout time.Duration, ns string) error {
	return retry(interval, int(timeout/interval), func() (bool, error) {
		listURI := fmt.Sprintf("/apis/%s/%s/namespaces/%s/canaries", spec.TPRGroup, spec.TPRVersion, ns)
		_, err := restcli.Get().RequestURI(listURI).DoRaw()
		if err != nil {
			if apierrors.IsNotFound(err) {
				// not set up yet. wait more.
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func newTPRClient() (*http.Client, error, string) {
	var masterURL string
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err, masterURL
	}

	cfg.GroupVersion = &schema.GroupVersion{
		Group:   spec.TPRGroup,
		Version: spec.TPRVersion,
	}
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	masterURL = cfg.Host

	restcli, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err, masterURL
	}
	return restcli.Client, nil, masterURL
}

func (c Controller) getListPath(ns string) string {
	return fmt.Sprintf("apis/%s/%s/namespaces/%s/canaries", spec.TPRGroup, spec.TPRVersion, ns)
}

func (c Controller) getCanaryList(restcli rest.Interface, ns string) (*spec.CanaryList, error) {
	b, err := restcli.Get().RequestURI(c.getListPath(ns)).DoRaw()
	if err != nil {
		return nil, err
	}

	canaries := &spec.CanaryList{}
	if err := json.Unmarshal(b, canaries); err != nil {
		return nil, err
	}
	return canaries, nil
}

func (c Controller) watchCanaries(watchVersion string) (<-chan *Event, <-chan error) {
	eventCh := make(chan *Event)
	errorCh := make(chan error)

	go func() {
		defer close(eventCh)

		for {
			// set up TPR client
			client, err, masterURL := newTPRClient()
			if err != nil {
				errorCh <- err
				return
			}

			// watch canaries
			resp, err := client.Get(fmt.Sprintf("%s/%s/?watch=true&resourceVersion=%s", masterURL, c.getListPath(c.config.Namespace), watchVersion))

			// check errors
			if err != nil {
				errorCh <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				errorCh <- fmt.Errorf("Invalid status code: %d", resp.StatusCode)
			}

			fmt.Printf("start watching at %v\n", watchVersion)

			// decode
			decoder := json.NewDecoder(resp.Body)
			for {
				// create event
				ev, st, err := pollEvent(decoder)

				// check errors
				if err != nil {
					if err == io.EOF {
						// apiserver will close stream periodically
						fmt.Println("apiserver closed stream")
						break
					}

					err := fmt.Errorf("received invalid event from API server: %v", err)
					errorCh <- err
					return
				}

				if st != nil {
					resp.Body.Close()

					if st.Code == http.StatusGone {
						// event history is outdated.
						// if nothing has changed, we can go back to watch again.
						canaryList, err := c.getCanaryList(c.config.KubeCli.CoreV1().RESTClient(), c.config.Namespace)
						if err == nil && !c.isCanariesCacheStale(canaryList.Items) {
							watchVersion = canaryList.Metadata.ResourceVersion
							break
						}

						// if anything has changed (or error on relist), we have to rebuild the state.
						// go to recovery path
						errorCh <- errors.New("requested version is outdated in apiserver")
						return
					}

					fmt.Printf("unexpected status response from API server: %v\n", st.Message)
				}

				// check status
				fmt.Printf("canary event: %v %v", ev.Type, ev.Object.Spec)

				// update watchVersion
				watchVersion = ev.Object.Metadata.ResourceVersion

				// add event to channel
				eventCh <- ev
			}

			// close body
			resp.Body.Close()
		}
	}()

	return eventCh, errorCh
}

func (c *Controller) isCanariesCacheStale(currentCanaries []spec.Canary) bool {
	if len(c.canaryRVs) != len(currentCanaries) {
		return true
	}

	for _, cc := range currentCanaries {
		rv, ok := c.canaryRVs[cc.Metadata.Name]
		if !ok || rv != cc.Metadata.ResourceVersion {
			return true
		}
	}

	return false
}

func (c Controller) handleCanaryEvent(event *Event) error {
	canarySpec := event.Object
	canaryName := canarySpec.Metadata.Name

	switch event.Type {
	case watch.Added:
		newCanary := canary.New(canarySpec)
		c.canaries[canaryName] = newCanary
		c.canaryRVs[canaryName] = canarySpec.Metadata.ResourceVersion

	case watch.Modified:
		if _, ok := c.canaries[canaryName]; !ok {
			return fmt.Errorf("unsafe state. canary was never created but we received event (%s)", event.Type)
		}
		c.canaries[canaryName].Update(canarySpec)
		c.canaryRVs[canaryName] = canarySpec.Metadata.ResourceVersion

	case watch.Deleted:
		if _, ok := c.canaries[canaryName]; !ok {
			return fmt.Errorf("unsafe state. canary was never created but we received event (%s)", event.Type)
		}
		c.canaries[canaryName].Delete()
		delete(c.canaries, canaryName)
		delete(c.canaryRVs, canaryName)

	}

	return nil
}

type rawEvent struct {
	Type   watch.EventType
	Object json.RawMessage
}

func pollEvent(decoder *json.Decoder) (*Event, *metav1.Status, error) {
	re := &rawEvent{}
	err := decoder.Decode(re)
	if err != nil {
		if err == io.EOF {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("fail to decode raw event from apiserver (%v)", err)
	}

	if re.Type == watch.Error {
		status := &metav1.Status{}
		err = json.Unmarshal(re.Object, status)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to decode (%s) into metav1.Status (%v)", re.Object, err)
		}
		return nil, status, nil
	}

	ev := &Event{
		Type:   re.Type,
		Object: &spec.Canary{},
	}
	err = json.Unmarshal(re.Object, ev.Object)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to unmarshal Canary object from data (%s): %v", re.Object, err)
	}
	return ev, nil, nil
}
