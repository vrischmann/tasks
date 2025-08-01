# Project Plan: Refactor `tasks` TUI to a Composable CLI Tool

**Objective:** Transform the existing Go TUI application, which uses the Bubble Tea framework, into a standard command-line interface (CLI) tool. The new tool will be stateless, composable, and follow standard Unix conventions, making it suitable for scripting and integration with other tools like `fzf` and `fish`.

**Guiding Principles:**

1.  **Stateless Commands:** Each command will initialize, perform a single action on the markdown file, and exit.
2.  **Composability:** The primary output command (`ls`) will produce clean, parsable text suitable for piping to other commands (`grep`, `awk`, `fzf`).
3.  **Leverage External Tools:** Delegate complex interactions. Use `$EDITOR` for editing and rely on shell functions/tools like `fzf` for interactivity.
4.  **Code Reuse:** Retain the core logic for parsing and writing the markdown file. The `Item` struct and `parseMarkdownFile` function are valuable and should be kept.
5.  **Dependency Reduction:** Remove the `bubbletea` dependency entirely. `lipgloss` can be retained for optional colored output but is no longer essential for layout.

---

### **Phase 1: Project Restructuring and Dependency Cleanup**

**Goal:** Strip out the TUI and prepare the foundation for the CLI.

1.  **Remove TUI Dependencies:**
    * In `go.mod`, remove the `github.com/charmbracelet/bubbletea` dependency.
    * Run `go mod tidy` to clean up the `go.sum` file.

2.  **Delete TUI-Specific Code from `main.go`:**
    * Delete the `Model` struct. It represents the TUI state and is no longer needed.
    * Delete all methods associated with the `Model` struct: `Init`, `Update`, `View`, and all their handlers (`handleNavigation`, `handleInputMode`, etc.).
    * Delete all rendering functions: `renderBanner`, `renderVisibleItems`, `renderInput`, `renderFooter`, `renderHelpScreen`.
    * Delete all `lipgloss` style variables that are related to the TUI layout (e.g., `selectedStyle`, `footerFilenameStyle`, `inputStyle`). Retain basic color styles like `successColor`, `mutedColor`, `accentColor` for potential use in the `ls` command's output.

3.  **Refactor `main.go` Structure:**
    * The file should now contain only the following key components:
        * `ItemType` enum (`TypeSection`, `TypeTask`).
        * `Item` struct.
        * `parseMarkdownFile()` function.
        * `saveToFile()` function (this will need to be adapted).
        * `getVersion()` function.
        * The `main()` function, which will be the new entry point for command parsing.

---

### **Phase 2: Implement the Core CLI Logic and the `ls` Command**

**Goal:** Create the main command dispatcher and the primary read-only command, `ls`.

1.  **Introduce a CLI Framework:**
    * Use the standard library's `flag` package for simplicity.

2.  **Modify the `main()` function:**
    * The `main` function will now act as a command dispatcher.
    * It should check `os.Args` to determine which subcommand is being called (`ls`, `add`, `done`, etc.).
    * Implement a `switch` statement on the subcommand.
    * A global `-f` or `--file` flag should be used to specify the markdown file path.

    **Example `main` function structure:**
    ```go
    func main() {
        // Define a global file flag
        filePath := flag.String("file", "TODO.md", "Path to the markdown file")
        flag.Parse()

        args := flag.Args()
        if len(args) < 1 {
            // Print usage info
            fmt.Println("Usage: tasks -file <path> <command> [args]")
            os.Exit(1)
        }

        cmd := args[0]
        cmdArgs := args[1:]

        switch cmd {
        case "ls":
            handleList(*filePath)
        case "add":
            handleAdd(*filePath, cmdArgs)
        // ... other cases
        default:
            fmt.Printf("Unknown command: %s\n", cmd)
            os.Exit(1)
        }
    }
    ```

3.  **Implement the `ls` Command Handler:**
    * Create a new function: `handleList(filePath string)`.
    * Inside `handleList`, call `parseMarkdownFile(filePath)` to get the `[]Item`.
    * Iterate through the `items` slice. For each item, print a formatted string to `stdout`.
    * **Crucially, prefix each line with its 1-based index.** This index is the "identifier" that other commands will use.
    * Use `lipgloss` subtly for color if desired (e.g., color for section headers, strikethrough for completed tasks). The output must remain easily parsable.

    **`ls` Output Format Example:**
    ```
    1   # Project Tasks
    2     - [ ] Setup React project
    3     - [x] Create main components
    4   ## UI Components
    5       - [ ] Button component
    ```

---

### **Phase 3: Implement Modifying Commands**

**Goal:** Implement the commands that read, modify, and save the markdown file.

1.  **Create a Centralized Data Handler:**
    * Create a struct, e.g., `TaskManager`, that holds the file path and the list of items.
    * `type TaskManager struct { FilePath string; Items []Item }`
    * Add methods to this struct: `Load()`, `Save()`, and methods for each action. This encapsulates the file I/O.

2.  **Implement `done` and `undo` Commands:**
    * Create `handleDone(filePath string, args []string)` and `handleUndo(...)`.
    * These functions will:
        a.  Parse the task identifier (the line number) from the arguments. Convert it to a 0-based index.
        b.  Create a `TaskManager` instance and call `tm.Load()`.
        c.  Access `tm.Items[index]`, check if it's a `TypeTask`, and modify its `Checked` status.
        d.  Call `tm.Save()`.

3.  **Implement `add` Command:**
    * Create `handleAdd(filePath string, args []string)`.
    * The function should parse a flag to indicate if the new item is a section, e.g., `--section` (or `-s`) which accepts an integer from 1 to 6.
    * It should also support a flag like `--after <id>` for positioning.
    * **Logic:**
        a.  Load items using `TaskManager`.
        b.  Check for the `--section` flag.
        c.  **If `--section` is present:** Create a new `Item` with `Type: TypeSection`, the specified `Level`, and `Checked: nil`.
        d.  **If `--section` is NOT present:** Create a new `Item` with `Type: TypeTask`, an appropriate `Level` (e.g., based on the item it's after), and `Checked` initialized to `false`.
        e.  Insert the new item into the `tm.Items` slice (either at the end or at the `--after` position).
        f.  Save the file.

4.  **Implement `rm` Command:**
    * Create `handleRemove(filePath string, args []string)`.
    * **Logic:**
        a.  Load items.
        b.  Find the item at the specified index.
        c.  **Important:** If the item is a `TypeSection`, you must also remove all of its children. Reuse the logic from the TUI's `deleteItem` function.
        d.  Remove the item(s) from the `tm.Items` slice and save the file.

5.  **Implement `edit` Command:**
    * Create `handleEdit(filePath string, args []string)`.
    * This command does not modify the file content directly in Go.
    * **Logic:**
        a.  Parse the task identifier.
        b.  **Modify `parseMarkdownFile` to store the original line number in the `Item` struct.** This is critical for finding the exact line to edit.
        c.  Find the line number in the file that corresponds to the item identifier.
        d.  Get the user's editor from the `$EDITOR` environment variable (defaulting to `vi` or `nano`).
        e.  Construct the command to execute (e.g., `vim +<line_number> <file_path>`).
        f.  Use `os/exec` to run the editor command, inheriting `stdin`, `stdout`, and `stderr`.

---

### **Phase 4: Finalization and Testing**

**Goal:** Ensure the CLI tool is robust and user-friendly.

1.  **Add Usage and Help Text:**
    * For each command, and for the main entry point, print clear usage instructions if the arguments are incorrect. The `flag` package provides default help text generation.

2.  **Refactor `main_test.go`:**
    * Delete all `teatest` tests, as they are for the TUI.
    * Write new unit tests for the CLI logic.
    * For each command handler (`handleDone`, `handleAdd`, etc.):
        a.  Create a temporary test file using the existing `createTestFile` helper.
        b.  Run the handler function with test arguments.
        c.  Read the content of the temporary file and assert that the changes are correct.
