package widgets

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// maxLSPMessage bounds one JSON-RPC message body from a language server. A
// server that announces more than this is treated as broken rather than allowed
// to allocate it.
const maxLSPMessage = 32 << 20

type LSPClient struct {
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	stderr          io.ReadCloser
	reader          *bufio.Scanner
	mu              sync.Mutex
	requestID       int
	handlers        map[int]chan []byte
	capabilities    map[string]interface{}
	connected       bool
	langID          string
	serverPath      string
	rootURI         string
	docVersion      int
	onDiagnostics   func(uri string, diagnostics []Diagnostic)
	onCompletion    func(reqID int, items []CompletionItem)
	onHover         func(reqID int, contents string)
	onDefinition    func(reqID int, uri string, line, col int)
	onReferences    func(reqID int, locations []Location)
	onSignatureHelp func(reqID int, hints []ParameterHint)
	uri             string
}

type Location struct {
	URI       string
	Line      int
	Character int
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *struct {
		TriggerKind int `json:"triggerKind"`
	} `json:"context,omitempty"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      struct {
		IncludeDeclaration bool `json:"includeDeclaration"`
	} `json:"context"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type SignatureHelpParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type GoToDefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	} `json:"context"`
}

type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

type InitializeParams struct {
	ProcessID    int                `json:"processId"`
	RootURI      string             `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace"`
	Window       WindowClientCapabilities       `json:"window"`
}

type TextDocumentClientCapabilities struct {
	Synchronization struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
		WillSave            bool `json:"willSave"`
		WillSaveWaitUntil   bool `json:"willSaveWaitUntil"`
		DidSave             bool `json:"didSave"`
	} `json:"synchronization"`
	Completion struct {
		DynamicRegistration bool     `json:"dynamicRegistration"`
		CompletionItem      struct{} `json:"completionItem"`
	} `json:"completion"`
	Hover struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"hover"`
	SignatureHelp struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"signatureHelp"`
	Declaration struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"declaration"`
	Definition struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"definition"`
	TypeDefinition struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"typeDefinition"`
	Implementation struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"implementation"`
	References struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"references"`
	DocumentHighlight struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"documentHighlight"`
	DocumentSymbol struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"documentSymbol"`
	Formatting struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"formatting"`
	RangeFormatting struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"rangeFormatting"`
	OnTypeFormatting struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"onTypeFormatting"`
	Rename struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"rename"`
	CodeAction struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"codeAction"`
}

type WorkspaceClientCapabilities struct {
	Symbol struct {
		DynamicRegistration bool `json:"dynamicRegistration"`
	} `json:"symbol"`
}

type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress"`
}

type LSPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewLSPClient(serverPath, langID, rootURI string) *LSPClient {
	return &LSPClient{
		serverPath:   serverPath,
		langID:       langID,
		rootURI:      rootURI,
		requestID:    1,
		handlers:     make(map[int]chan []byte),
		capabilities: make(map[string]interface{}),
		docVersion:   1,
	}
}

func (c *LSPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	cmd := exec.Command(c.serverPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.reader = bufio.NewScanner(stdout)
	c.reader.Buffer(make([]byte, 0, 65536), 1024*1024)
	c.connected = true

	go c.readStderr()
	go c.readLoop()

	initParams := InitializeParams{
		ProcessID: 0,
		RootURI:   c.rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Synchronization: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
					WillSave            bool `json:"willSave"`
					WillSaveWaitUntil   bool `json:"willSaveWaitUntil"`
					DidSave             bool `json:"didSave"`
				}{
					DynamicRegistration: true,
					WillSave:            true,
					DidSave:             true,
				},
				Completion: struct {
					DynamicRegistration bool     `json:"dynamicRegistration"`
					CompletionItem      struct{} `json:"completionItem"`
				}{
					DynamicRegistration: true,
				},
				Hover: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				SignatureHelp: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				Definition: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				References: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				DocumentSymbol: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				CodeAction: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
				Rename: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
			},
			Workspace: WorkspaceClientCapabilities{
				Symbol: struct {
					DynamicRegistration bool `json:"dynamicRegistration"`
				}{
					DynamicRegistration: true,
				},
			},
		},
	}

	resp, err := c.sendRequest("initialize", initParams)
	if err != nil {
		c.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	var initResult struct {
		Capabilities map[string]interface{} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp, &initResult); err != nil {
		c.Close()
		return fmt.Errorf("parse initialize result: %w", err)
	}
	c.capabilities = initResult.Capabilities

	c.sendNotification("initialized", struct{}{})
	return nil
}

func (c *LSPClient) readStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
	}
}

func (c *LSPClient) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			c.connected = false
			return
		}

		var contentLength int
		_, scanErr := fmt.Sscanf(line, "Content-Length: %d", &contentLength)
		// The server is a separate process: its framing is input, not a promise.
		// A negative or enormous length would desync or exhaust the reader.
		if scanErr == nil && contentLength < 0 {
			c.connected = false
			return
		}
		if scanErr == nil && contentLength > maxLSPMessage {
			c.connected = false
			return
		}
		if line == "\r\n" && contentLength > 0 {
			body := make([]byte, contentLength)
			_, err := io.ReadFull(reader, body)
			if err != nil {
				c.connected = false
				return
			}
			c.handleMessage(body)
			contentLength = 0
		}
	}
}

func (c *LSPClient) handleMessage(body []byte) {
	var msg struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id,omitempty"`
		Method  string          `json:"method,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}

	if msg.Method != "" {
		switch msg.Method {
		case "textDocument/publishDiagnostics":
			var params struct {
				URI         string       `json:"uri"`
				Diagnostics []Diagnostic `json:"diagnostics"`
			}
			if json.Unmarshal(msg.Params, &params) == nil {
				if c.onDiagnostics != nil {
					c.onDiagnostics(params.URI, params.Diagnostics)
				}
			}
		}
		return
	}

	var id int
	if len(msg.ID) > 0 {
		json.Unmarshal(msg.ID, &id)
	}

	c.mu.Lock()
	ch, ok := c.handlers[id]
	c.mu.Unlock()

	if ok {
		ch <- body
	}
}

func (c *LSPClient) sendRequest(method string, params interface{}) ([]byte, error) {
	c.mu.Lock()
	id := c.requestID
	c.requestID++
	ch := make(chan []byte, 1)
	c.handlers[id] = ch
	c.mu.Unlock()

	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()
		return nil, err
	}

	_, err = fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(data), data)
	if err != nil {
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()

		var rmsg struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resp, &rmsg); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if rmsg.Error != nil {
			return nil, fmt.Errorf("LSP error %d: %s", rmsg.Error.Code, rmsg.Error.Message)
		}
		return rmsg.Result, nil

	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.handlers, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for LSP response")
	}
}

func (c *LSPClient) sendNotification(method string, params interface{}) {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin != nil {
		fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n%s", len(data), data)
	}
}

func (c *LSPClient) DidOpen(uri, languageID, text string) {
	c.uri = uri
	c.langID = languageID
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    c.docVersion,
			Text:       text,
		},
	}
	c.sendNotification("textDocument/didOpen", params)
}

func (c *LSPClient) DidChange(text string) {
	c.docVersion++
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     c.uri,
			Version: c.docVersion,
		},
		ContentChanges: []struct {
			Text string `json:"text"`
		}{
			{Text: text},
		},
	}
	c.sendNotification("textDocument/didChange", params)
}

func (c *LSPClient) DidClose() {
	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
	}
	c.sendNotification("textDocument/didClose", params)
}

func (c *LSPClient) DidSave() {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Text         string                 `json:"text,omitempty"`
	}{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
	}
	c.sendNotification("textDocument/didSave", params)
}

func (c *LSPClient) Completion(line, col int) (int, []CompletionItem, error) {
	params := CompletionParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
	}

	data, err := c.sendRequest("textDocument/completion", params)
	if err != nil {
		return 0, nil, err
	}

	var result struct {
		IsIncomplete bool              `json:"isIncomplete"`
		Items        []json.RawMessage `json:"items"`
	}

	var items []json.RawMessage
	if err := json.Unmarshal(data, &result); err == nil {
		items = result.Items
	} else if err := json.Unmarshal(data, &items); err != nil {
		return 0, nil, fmt.Errorf("parse completion: %w", err)
	}

	completions := make([]CompletionItem, 0, len(items))
	for _, raw := range items {
		var item struct {
			Label      string `json:"label"`
			Kind       int    `json:"kind"`
			Detail     string `json:"detail"`
			InsertText string `json:"insertText"`
		}
		if json.Unmarshal(raw, &item) == nil {
			completions = append(completions, CompletionItem{
				Label:      item.Label,
				Kind:       item.Kind,
				Detail:     item.Detail,
				InsertText: item.InsertText,
			})
		}
	}

	return 0, completions, nil
}

func (c *LSPClient) Hover(line, col int) (string, error) {
	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
	}

	data, err := c.sendRequest("textDocument/hover", params)
	if err != nil {
		return "", err
	}

	var result struct {
		Contents struct {
			Kind  string `json:"kind"`
			Value string `json:"value"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		var oldFormat []struct {
			Language string `json:"language"`
			Value    string `json:"value"`
		}
		if json.Unmarshal(data, &oldFormat) == nil && len(oldFormat) > 0 {
			return oldFormat[0].Value, nil
		}
		var strContent string
		if json.Unmarshal(data, &strContent) == nil {
			return strContent, nil
		}
		return "", nil
	}

	return result.Contents.Value, nil
}

func (c *LSPClient) GoToDefinition(line, col int) (string, Position, error) {
	params := GoToDefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
	}

	data, err := c.sendRequest("textDocument/definition", params)
	if err != nil {
		return "", Position{}, err
	}

	var location struct {
		URI   string `json:"uri"`
		Range struct {
			Start Position `json:"start"`
		} `json:"range"`
	}

	if err := json.Unmarshal(data, &location); err != nil {
		var locations []struct {
			URI   string `json:"uri"`
			Range struct {
				Start Position `json:"start"`
			} `json:"range"`
		}
		if json.Unmarshal(data, &locations) == nil && len(locations) > 0 {
			location = locations[0]
		} else {
			return "", Position{}, fmt.Errorf("parse definition: %w", err)
		}
	}

	return location.URI, location.Range.Start, nil
}

func (c *LSPClient) FindReferences(line, col int) ([]Location, error) {
	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
	}

	params.Context.IncludeDeclaration = true

	data, err := c.sendRequest("textDocument/references", params)
	if err != nil {
		return nil, err
	}

	var locations []struct {
		URI   string `json:"uri"`
		Range struct {
			Start Position `json:"start"`
			End   Position `json:"end"`
		} `json:"range"`
	}

	if err := json.Unmarshal(data, &locations); err != nil {
		return nil, fmt.Errorf("parse references: %w", err)
	}

	result := make([]Location, len(locations))
	for i, loc := range locations {
		result[i] = Location{
			URI:       loc.URI,
			Line:      loc.Range.Start.Line,
			Character: loc.Range.Start.Character,
		}
	}

	return result, nil
}

func (c *LSPClient) SignatureHelp(line, col int) ([]ParameterHint, error) {
	params := SignatureHelpParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
	}

	data, err := c.sendRequest("textDocument/signatureHelp", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Signatures []struct {
			Label         string `json:"label"`
			Documentation string `json:"documentation"`
			Parameters    []struct {
				Label         string `json:"label"`
				Documentation string `json:"documentation"`
			} `json:"parameters"`
		} `json:"signatures"`
		ActiveSignature int `json:"activeSignature"`
		ActiveParameter int `json:"activeParameter"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse signature help: %w", err)
	}

	hints := make([]ParameterHint, 0, len(result.Signatures))
	for _, sig := range result.Signatures {
		hints = append(hints, ParameterHint{
			Label:         sig.Label,
			Documentation: sig.Documentation,
			ActiveParam:   result.ActiveParameter,
		})
	}

	return hints, nil
}

func (c *LSPClient) RenameSymbol(line, col int, newName string) error {
	params := RenameParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Position:     Position{Line: line, Character: col},
		NewName:      newName,
	}

	resp, err := c.sendRequest("textDocument/rename", params)
	if err != nil {
		return err
	}

	var changes struct {
		DocumentChanges []struct {
			TextDocument struct {
				URI     string `json:"uri"`
				Version int    `json:"version"`
			} `json:"textDocument"`
			Edits []struct {
				Range   Range  `json:"range"`
				NewText string `json:"newText"`
			} `json:"edits"`
		} `json:"documentChanges"`
	}

	if json.Unmarshal(resp, &changes) != nil {
		var altChanges struct {
			Changes map[string][]struct {
				Range   Range  `json:"range"`
				NewText string `json:"newText"`
			} `json:"changes"`
		}
		if json.Unmarshal(resp, &altChanges) != nil {
			return fmt.Errorf("parse rename result")
		}
	}

	return nil
}

func (c *LSPClient) DocumentSymbols() ([]interface{}, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
	}

	data, err := c.sendRequest("textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}

	var symbols []interface{}
	if err := json.Unmarshal(data, &symbols); err != nil {
		return nil, fmt.Errorf("parse document symbols: %w", err)
	}

	return symbols, nil
}

func (c *LSPClient) CodeAction(line, col int, diagnostics []Diagnostic) ([]interface{}, error) {
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: c.uri},
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col + 1},
		},
	}
	params.Context.Diagnostics = diagnostics

	data, err := c.sendRequest("textDocument/codeAction", params)
	if err != nil {
		return nil, err
	}

	var actions []interface{}
	if err := json.Unmarshal(data, &actions); err != nil {
		return nil, fmt.Errorf("parse code actions: %w", err)
	}

	return actions, nil
}

func (c *LSPClient) SetOnDiagnostics(handler func(uri string, diagnostics []Diagnostic)) {
	c.onDiagnostics = handler
}

func (c *LSPClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}
	c.connected = false

	c.DidClose()

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	for id, ch := range c.handlers {
		close(ch)
		delete(c.handlers, id)
	}
}

func LangIDForFile(path string) string {
	ext := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			ext = path[i:]
			break
		}
	}
	switch ext {
	case ".go":
		return "go"
	case ".py", ".pyw":
		return "python"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h", ".hpp":
		return "c"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".dart":
		return "dart"
	case ".html", ".htm":
		return "html"
	case ".css", ".scss", ".less":
		return "css"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "shellscript"
	case ".pl":
		return "perl"
	case ".lua":
		return "lua"
	case ".r":
		return "r"
	}
	return "plaintext"
}

func ServerForLanguage(langID string) string {
	servers := map[string]string{
		"go":         "gopls",
		"python":     "pylsp",
		"javascript": "typescript-language-server --stdio",
		"typescript": "typescript-language-server --stdio",
		"rust":       "rust-analyzer",
		"java":       "eclipse.jdt.ls",
		"cpp":        "clangd",
		"c":          "clangd",
		"csharp":     "omnisharp",
		"ruby":       "solargraph",
		"php":        "phpactor",
		"swift":      "sourcekit-lsp",
		"kotlin":     "kotlin-language-server",
		"dart":       "dart",
		"html":       "vscode-html-language-server --stdio",
		"css":        "vscode-css-language-server --stdio",
		"json":       "vscode-json-language-server --stdio",
		"yaml":       "yaml-language-server --stdio",
		"markdown":   "markdown-language-server --stdio",
		"sql":        "sql-language-server",
		"bash":       "bash-language-server",
		"lua":        "lua-language-server",
	}
	return servers[langID]
}
