package docmost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"strings"
	"time"
)

// Client is the Docmost API client
type Client struct {
	baseURL    string
	email      string
	password   string
	httpClient *http.Client
	loggedIn   bool
}

// Space represents a Docmost space
type Space struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Page represents a Docmost page
type Page struct {
	ID           string  `json:"id"`
	SlugID       string  `json:"slugId"`
	Title        string  `json:"title"`
	Icon         *string `json:"icon"`
	Position     string  `json:"position"`
	ParentPageID *string `json:"parentPageId"`
	SpaceID      string  `json:"spaceId"`
	CreatorID    string  `json:"creatorId"`
	HasChildren  bool    `json:"hasChildren"`
}

// PageMeta represents metadata for a single page
type PageMeta struct {
	ID           string      `json:"id"`
	SlugID       string      `json:"slugId"`
	Title        string      `json:"title"`
	Icon         *string     `json:"icon,omitempty"`
	Position     string      `json:"position"`
	ParentPageID *string     `json:"parentPageId,omitempty"`
	HasChildren  bool        `json:"hasChildren"`
	Children     []*PageMeta `json:"children,omitempty"`
	FilePath     string      `json:"filePath,omitempty"`
}

// SpaceMeta represents metadata for a space including page tree structure
type SpaceMeta struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
	Pages       []*PageMeta `json:"pages"`
	TotalPages  int         `json:"totalPages"`
}

// APIResponse is the generic API response wrapper
type APIResponse struct {
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
	Status  int             `json:"status"`
}

// SpaceListData represents the spaces list response data
type SpaceListData struct {
	Items []Space `json:"items"`
	Meta  struct {
		Limit       int  `json:"limit"`
		Page        int  `json:"page"`
		HasNextPage bool `json:"hasNextPage"`
		HasPrevPage bool `json:"hasPrevPage"`
	} `json:"meta"`
}

// PageListData represents the pages list response data
type PageListData struct {
	Items []Page `json:"items"`
	Meta  struct {
		Limit       int  `json:"limit"`
		Page        int  `json:"page"`
		HasNextPage bool `json:"hasNextPage"`
		HasPrevPage bool `json:"hasPrevPage"`
	} `json:"meta"`
}

// NewClient creates a new Docmost API client with cookie-based authentication
func NewClient(baseURL, email, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		email:    email,
		password: password,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Jar:     jar,
		},
		loggedIn: false,
	}, nil
}

// Login authenticates with Docmost API
func (c *Client) Login() error {
	loginData := map[string]string{
		"email":    c.email,
		"password": c.password,
	}

	body, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	url := fmt.Sprintf("%s/api/auth/login", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	c.loggedIn = true
	return nil
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	if !c.loggedIn {
		if err := c.Login(); err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}
	}

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// ListSpaces retrieves all accessible spaces
func (c *Client) ListSpaces() ([]Space, error) {
	reqBody := map[string]interface{}{
		"limit":  100,
		"offset": 0,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := c.doRequest(http.MethodPost, "/api/spaces/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list spaces failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var data SpaceListData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to decode spaces data: %w", err)
	}

	return data.Items, nil
}

// ListSidebarPages retrieves all pages in a space from sidebar
func (c *Client) ListSidebarPages(spaceID string) ([]Page, error) {
	reqBody := map[string]interface{}{
		"spaceId": spaceID,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := c.doRequest(http.MethodPost, "/api/pages/sidebar-pages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list pages failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var data PageListData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to decode pages data: %w", err)
	}

	return data.Items, nil
}

// ListChildPages retrieves child pages of a specific page
func (c *Client) ListChildPages(pageID string) ([]Page, error) {
	reqBody := map[string]interface{}{
		"pageId": pageID,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := c.doRequest(http.MethodPost, "/api/pages/sidebar-pages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list child pages failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var data PageListData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to decode child pages data: %w", err)
	}

	return data.Items, nil
}

// GetSpaceMetadata retrieves metadata for a space including page tree structure
func (c *Client) GetSpaceMetadata(space Space, files map[string][]byte) (*SpaceMeta, error) {
	pages, err := c.ListSidebarPages(space.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pages: %w", err)
	}

	// Build root pages with recursive children fetch
	var rootPages []*PageMeta
	totalPages := 0
	for _, p := range pages {
		pm := c.buildPageMeta(p, files, &totalPages)
		rootPages = append(rootPages, pm)
	}

	// Sort pages by position
	sortPagesByPosition(rootPages)

	return &SpaceMeta{
		ID:          space.ID,
		Name:        space.Name,
		Slug:        space.Slug,
		Description: space.Description,
		CreatedAt:   space.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   space.UpdatedAt.Format(time.RFC3339),
		Pages:       rootPages,
		TotalPages:  totalPages,
	}, nil
}

// buildPageMeta builds PageMeta recursively fetching children
func (c *Client) buildPageMeta(p Page, files map[string][]byte, totalCount *int) *PageMeta {
	*totalCount++

	pm := &PageMeta{
		ID:           p.ID,
		SlugID:       p.SlugID,
		Title:        p.Title,
		Icon:         p.Icon,
		Position:     p.Position,
		ParentPageID: p.ParentPageID,
		HasChildren:  p.HasChildren,
		Children:     []*PageMeta{},
		FilePath:     findFilePathForPage(p.Title, files),
	}

	// Recursively fetch children if page has children
	if p.HasChildren {
		childPages, err := c.ListChildPages(p.ID)
		if err == nil {
			for _, child := range childPages {
				childMeta := c.buildPageMeta(child, files, totalCount)
				pm.Children = append(pm.Children, childMeta)
			}
			sortPagesByPosition(pm.Children)
		}
	}

	return pm
}

// findFilePathForPage tries to find the file path for a page based on title
func findFilePathForPage(title string, files map[string][]byte) string {
	sanitized := SanitizeFilename(title)
	for path := range files {
		// Check if this is a markdown file that matches the title
		if strings.HasSuffix(path, ".md") {
			base := strings.TrimSuffix(filepath.Base(path), ".md")
			if base == sanitized || base == title {
				return path
			}
		}
	}
	return ""
}

// sortPagesByPosition sorts pages by their position field
func sortPagesByPosition(pages []*PageMeta) {
	// Simple bubble sort - position is a string that can be compared lexicographically
	for i := 0; i < len(pages); i++ {
		for j := i + 1; j < len(pages); j++ {
			if pages[i].Position > pages[j].Position {
				pages[i], pages[j] = pages[j], pages[i]
			}
		}
	}
}
