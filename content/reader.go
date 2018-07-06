package content

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/Financial-Times/uuid-utils-go"
	"github.com/pkg/errors"
)

const (
	userAgent      = "User-Agent"
	userAgentValue = "UPP_content-unroller"
)

type Reader interface {
	Get([]string, string) (map[string]Content, error)
	GetInternal([]string, string) (map[string]Content, error)
	GetPreview([]string, string) (map[string]Content, error)
	GetInternalPreview([]string, string) (map[string]Content, error)
}

type ReaderFunc func([]string, string) (map[string]Content, error)

type ReaderConfig struct {
	ContentStoreAppName           string
	ContentStoreHost              string
	ContentStorePath              string
	ContentStoreInternalPath      string
	ContentPreviewAppName         string
	ContentPreviewHost            string
	ContentPreviewPath            string
	InternalContentPreviewAppName string
	InternalContentPreviewHost    string
	InternalContentPreviewPath    string
}

type ContentReader struct {
	client *http.Client
	config ReaderConfig
}

func NewContentReader(rConfig ReaderConfig, client *http.Client) *ContentReader {
	return &ContentReader{
		client: client,
		config: rConfig,
	}
}

// Get content from content-public-read
func (cr *ContentReader) Get(uuids []string, tid string) (map[string]Content, error) {
	var cm = make(map[string]Content)
	requestURL := fmt.Sprintf("%s%s", cr.config.ContentStoreHost, cr.config.ContentStorePath)

	contentBatch, err := cr.doGet(uuids, tid, requestURL, cr.config.ContentStoreAppName)
	if err != nil {
		return cm, err
	}

	var imgModelUUIDs []string
	for _, c := range contentBatch {
		cr.addItemToMap(c, cm)
		if _, foundMembers := c[members]; foundMembers {
			imgModelUUIDs = append(imgModelUUIDs, c.getMembersUUID()...)
		}
	}

	if len(imgModelUUIDs) == 0 {
		return cm, nil
	}

	imgModelsList, err := cr.doGet(imgModelUUIDs, tid, cr.config.ContentStoreHost, cr.config.ContentStoreAppName)
	if err != nil {
		return cm, err
	}

	for _, i := range imgModelsList {
		cr.addItemToMap(i, cm)
	}

	return cm, nil
}

// GetInternal internal components from document-store-api
func (cr *ContentReader) GetInternal(uuids []string, tid string) (map[string]Content, error) {
	var cm = make(map[string]Content)
	requestURL := fmt.Sprintf("%s%s", cr.config.ContentStoreHost, cr.config.ContentStoreInternalPath)

	internalContent, err := cr.doGet(uuids, tid, requestURL, cr.config.ContentStoreAppName)
	if err != nil {
		return cm, err
	}

	for _, c := range internalContent {
		cr.addItemToMap(c, cm)
	}

	return cm, nil
}

// GetPreview from Methode API
func (cr *ContentReader) GetPreview(uuids []string, tid string) (map[string]Content, error) {
	var cm = make(map[string]Content)

	for _, uuid := range uuids {
		requestURL := fmt.Sprintf("%s%s/%s", cr.config.ContentPreviewHost, cr.config.ContentPreviewPath, uuid)
		content, err := cr.doGetPreview(uuid, tid, requestURL, cr.config.ContentPreviewAppName)
		if err != nil {
			return nil, err
		}

		cm[uuid] = content
	}

	return cm, nil
}

// GetInternalPreview reads internalcomponents from Methode API
func (cr *ContentReader) GetInternalPreview(uuids []string, tid string) (map[string]Content, error) {
	var cm = make(map[string]Content)

	for _, uuid := range uuids {
		requestURL := fmt.Sprintf("%s%s/%s", cr.config.InternalContentPreviewHost, cr.config.InternalContentPreviewPath, uuid)
		content, err := cr.doGetPreview(uuid, tid, requestURL, cr.config.InternalContentPreviewAppName)
		if err != nil {
			return nil, err
		}

		cm[uuid] = content
	}

	return cm, nil
}

func (cr *ContentReader) doGet(uuids []string, tid string, reqURL string, appName string) ([]Content, error) {
	var cb []Content

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return cb, errors.Wrapf(err, "Error connecting to %v", appName)
	}

	req.Header.Add(transactionidutils.TransactionIDHeader, tid)
	req.Header.Set(userAgent, userAgentValue)
	q := req.URL.Query()
	for _, uuid := range uuids {
		if err = uuidutils.ValidateUUID(uuid); err == nil {
			q.Add("uuid", uuid)
		}
	}
	req.URL.RawQuery = q.Encode()

	res, err := cr.client.Do(req)
	if err != nil {
		return cb, errors.Wrapf(err, "Request to %v failed.", appName)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return cb, errors.Errorf("Request to %v failed with status code %d", appName, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return cb, errors.Wrapf(err, "Error reading response received from %v", appName)
	}

	err = json.Unmarshal(body, &cb)
	if err != nil {
		return cb, errors.Wrapf(err, "Error unmarshalling response from %v", appName)
	}
	return cb, nil
}

func (cr *ContentReader) doGetPreview(uuid string, tid string, reqURL string, appName string) (Content, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Error connecting to %v for uuid: %s", appName, uuid)
	}

	req.Header.Set(transactionidutils.TransactionIDHeader, tid)
	req.Header.Set("User-Agent", userAgentValue)

	res, err := cr.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Request to %v failed for uuid: %s", appName, uuid)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Request to %v failed for uuid: %s with status code %d", appName, uuid, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading response received from %v", appName)
	}

	var content Content
	err = json.Unmarshal(body, &content)
	if err != nil {
		return content, errors.Wrapf(err, "Error unmarshalling response from %v", appName)
	}
	return content, nil
}

func (cr *ContentReader) addItemToMap(c Content, cm map[string]Content) {
	id, ok := c[id].(string)
	if !ok {
		return
	}
	uuid, err := extractUUIDFromString(id)
	if err != nil {
		return
	}
	cm[uuid] = c
}
