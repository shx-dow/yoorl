package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shx-dow/yoorl/store"
	"github.com/stretchr/testify/assert"
)

func setupTest() *gin.Engine {
	store.SetStore(store.NewMemoryStore())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/urls", HandleListUrls)
	r.POST("/v1/urls", CreateShortUrl)
	r.DELETE("/v1/urls/:shortUrl", DeleteShortUrl)
	r.PUT("/v1/urls/:shortUrl", UpdateShortUrl)
	r.GET("/v1/urls/:shortUrl/analytics", HandleGetAnalytics)
	r.GET("/r/:shortUrl", HandleShortUrlRedirect)
	return r
}

func TestCreateShortUrl(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "short url created successfully", resp["message"])
}

func TestCreateShortUrlInvalidUrl(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "not-a-valid-url", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateShortUrlWithCustomAlias(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123", "custom_alias": "myLink"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateShortUrlCustomAliasCollision(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123", "custom_alias": "myLink"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req2 := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestHandleShortUrlRedirectNotFound(t *testing.T) {
	r := setupTest()

	req := httptest.NewRequest(http.MethodGet, "/r/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleShortUrlRedirectSuccess(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	shortUrl := data["short_url"].(string)
	shortUrl = shortUrl[strings.LastIndex(shortUrl, "/r/")+3:]

	req2 := httptest.NewRequest(http.MethodGet, "/r/"+shortUrl, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusFound, w2.Code)
	assert.Equal(t, "https://example.com", w2.Header().Get("Location"))
}

func TestDeleteShortUrl(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	shortUrl := data["short_url"].(string)
	shortUrl = shortUrl[strings.LastIndex(shortUrl, "/r/")+3:]

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/urls/"+shortUrl, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, delReq)
	assert.Equal(t, http.StatusOK, w2.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/r/"+shortUrl, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, getReq)
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestDeleteShortUrlNotFound(t *testing.T) {
	r := setupTest()

	req := httptest.NewRequest(http.MethodDelete, "/v1/urls/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnalyticsNotFound(t *testing.T) {
	r := setupTest()

	req := httptest.NewRequest(http.MethodGet, "/v1/urls/nonexistent/analytics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnalyticsZeroClicks(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	shortUrl := data["short_url"].(string)
	shortUrl = shortUrl[strings.LastIndex(shortUrl, "/r/")+3:]

	req2 := httptest.NewRequest(http.MethodGet, "/v1/urls/"+shortUrl+"/analytics", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data2 := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data2["total_clicks"])
}

func TestUpdateShortUrl(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	shortUrl := data["short_url"].(string)
	shortUrl = shortUrl[strings.LastIndex(shortUrl, "/r/")+3:]

	updateBody := `{"long_url": "https://updated.com"}`
	updateReq := httptest.NewRequest(http.MethodPut, "/v1/urls/"+shortUrl, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, updateReq)
	assert.Equal(t, http.StatusOK, w2.Code)

	redirectReq := httptest.NewRequest(http.MethodGet, "/r/"+shortUrl, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, redirectReq)
	assert.Equal(t, http.StatusFound, w3.Code)
	assert.Equal(t, "https://updated.com", w3.Header().Get("Location"))
}

func TestUpdateShortUrlNotFound(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://updated.com"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/urls/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnalyticsWithClicks(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	shortUrl := data["short_url"].(string)
	shortUrl = shortUrl[strings.LastIndex(shortUrl, "/r/")+3:]

	_ = store.RecordClick(shortUrl, store.ClickEvent{})
	_ = store.RecordClick(shortUrl, store.ClickEvent{})

	req2 := httptest.NewRequest(http.MethodGet, "/v1/urls/"+shortUrl+"/analytics", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data2 := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data2["total_clicks"])
}

func TestListUrls(t *testing.T) {
	r := setupTest()

	body := `{"long_url": "https://example.com", "user_id": "test123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/v1/urls", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)

	entry := data[0].(map[string]interface{})
	assert.Equal(t, "https://example.com", entry["long_url"])
	assert.Equal(t, "test123", entry["user_id"])
}

func TestListUrlsEmpty(t *testing.T) {
	r := setupTest()

	req := httptest.NewRequest(http.MethodGet, "/v1/urls", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 0)
}
