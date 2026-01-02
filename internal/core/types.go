package core

import "sync"

// JSFile represents a discovered JavaScript file
type JSFile struct {
	URL        string
	Filename   string
	LocalPath  string
	Downloaded bool
	Beautified bool
}

// Secret represents a found secret
type Secret struct {
	Type     string
	Value    string
	File     string
	Line     int
	Detector string // e.g. "gitleaks", "regex"
}

// Endpoint represents a found endpoint
type Endpoint struct {
	Path   string
	Method string
	File   string
	Source string // e.g. "linkfinder"
}

// PipelineState holds the data as it flows through stages
type PipelineState struct {
	URLs            []string
	RawJSFiles      []*JSFile
	BeautifiedFiles []string
	Secrets         []Secret
	Endpoints       []Endpoint

	// Locks for concurrent access
	Mu sync.RWMutex
}

func NewPipelineState() *PipelineState {
	return &PipelineState{
		URLs:       make([]string, 0),
		RawJSFiles: make([]*JSFile, 0),
		Secrets:    make([]Secret, 0),
		Endpoints:  make([]Endpoint, 0),
	}
}
