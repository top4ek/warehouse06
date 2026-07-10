package domain

import "time"

type EntryType string

const (
	EntryTypeDirectory EntryType = "directory"
	EntryTypeFile      EntryType = "file"
)

type Entry struct {
	ID              int64     `json:"id"`
	Path            string    `json:"path"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	DescriptionHTML string    `json:"description_html,omitempty"`
	ContentHTML     string    `json:"content_html"`
	Date            string    `json:"date"`
	Type            EntryType `json:"type"`
	Youtube         string    `json:"youtube,omitempty"`
	Platform        string    `json:"platform,omitempty"`
	PreviewImage    string    `json:"preview_image,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	Tags        []Tag       `json:"tags,omitempty"`
	Authors     []Author    `json:"authors,omitempty"`
	Screenshots []File      `json:"screenshots,omitempty"`
	Files       []File      `json:"files,omitempty"`
	Directories []Directory `json:"directories,omitempty"`
	Requires    []string    `json:"require,omitempty"` // Paths to other entries
}

type Author struct {
	ID            int64    `json:"id"`
	DirectoryName string   `json:"directory_name"`
	Name          string   `json:"name"`
	Address       string   `json:"address"`
	ContentHTML   string   `json:"content_html"`
	PreviewImage  string   `json:"preview_image,omitempty"`
	EntryCount    int      `json:"entry_count,omitempty"`
	Entries       []*Entry `json:"entries,omitempty"`
}

type EntryListResult struct {
	Items []*Entry `json:"items"`
	Total int      `json:"total"`
}

type Tag struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	EntryCount int    `json:"entry_count,omitempty"`
}

type File struct {
	ID       int64  `json:"id"`
	EntryID  int64  `json:"entry_id"`
	Filename string `json:"filename"`
	Filepath string `json:"filepath"`
	IsImage  bool   `json:"is_image"`
}

type Directory struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Platform struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ContentHTML string `json:"content_html,omitempty"`
	EntryCount  int    `json:"entry_count"`
}
