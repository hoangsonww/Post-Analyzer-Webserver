package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newPostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "posts",
		Short: "Manage posts (list, get, create, update, delete, analyze, export)",
	}
	cmd.AddCommand(
		newPostsListCmd(),
		newPostsGetCmd(),
		newPostsCreateCmd(),
		newPostsUpdateCmd(),
		newPostsDeleteCmd(),
		newPostsAnalyzeCmd(),
		newPostsReanalyzeCmd(),
		newPostsExportCmd(),
	)
	return cmd
}

// lastFieldRE finds the final tabwriter-padded column (2+ spaces, since
// that's tabwriter's minimum padding, followed by the field's content
// through end of line) so it can be colorized without disturbing the
// padding tabwriter already computed.
var lastFieldRE = regexp.MustCompile(`\s{2,}\S+$`)

func printPostsTable(posts []Post) {
	// Format with plain text first. tabwriter sizes columns from each
	// cell's byte length — coloring cells *before* handing them to it
	// would count the invisible ANSI escape bytes toward that width and
	// throw off alignment (confirmed by actually running this before
	// this fix existed). Coloring substrings of the already-aligned
	// plain output is safe: a terminal renders escape codes as
	// zero-width, so it doesn't affect spacing.
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tUSER\tTITLE\tCREATED")
	for _, p := range posts {
		_, _ = fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", p.ID, p.UserID, truncate(p.Title, 50), p.CreatedAt)
	}
	_ = w.Flush()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	for i, line := range lines {
		if i == 0 {
			fmt.Println(bold(line))
			continue
		}
		fmt.Println(colorizePostsRow(line))
	}
}

func colorizePostsRow(line string) string {
	idEnd := strings.IndexByte(line, ' ')
	if idEnd < 0 {
		return line
	}
	line = cyan(line[:idEnd]) + line[idEnd:]
	return lastFieldRE.ReplaceAllStringFunc(line, func(m string) string {
		trimmed := strings.TrimLeft(m, " ")
		return m[:len(m)-len(trimmed)] + dim(trimmed)
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func newPostsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List posts",
		RunE: func(cmd *cobra.Command, args []string) error {
			posts, err := clientFromFlags().ListPosts()
			if err != nil {
				return err
			}
			if len(posts) == 0 {
				fmt.Println(dim("No posts found."))
				return nil
			}
			printPostsTable(posts)
			return nil
		},
	}
}

func newPostsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a post by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			post, err := clientFromFlags().GetPost(args[0])
			if err != nil {
				return err
			}
			return printJSON(post)
		},
	}
}

func newPostsCreateCmd() *cobra.Command {
	var userID int
	var title, body string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a post",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" || body == "" {
				return fmt.Errorf("--title and --body are required")
			}
			post, err := clientFromFlags().CreatePost(userID, title, body)
			if err != nil {
				return err
			}
			fmt.Println(ok(fmt.Sprintf("Created post %s", cyan(fmt.Sprintf("#%d", post.ID)))))
			return printJSON(post)
		},
	}
	c.Flags().IntVar(&userID, "user-id", 1, "author user ID")
	c.Flags().StringVar(&title, "title", "", "post title (required)")
	c.Flags().StringVar(&body, "body", "", "post body (required)")
	return c
}

func newPostsUpdateCmd() *cobra.Command {
	var title, body string
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			post, err := clientFromFlags().UpdatePost(args[0], title, body)
			if err != nil {
				return err
			}
			fmt.Println(ok(fmt.Sprintf("Updated post %s", cyan(fmt.Sprintf("#%d", post.ID)))))
			return printJSON(post)
		},
	}
	c.Flags().StringVar(&title, "title", "", "new title")
	c.Flags().StringVar(&body, "body", "", "new body")
	return c
}

func newPostsDeleteCmd() *cobra.Command {
	var mfa bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a post (editors need --mfa; admins don't)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := clientFromFlags().DeletePost(args[0], mfa); err != nil {
				return err
			}
			fmt.Println(ok(fmt.Sprintf("Deleted post %s", cyan("#"+args[0]))))
			return nil
		},
	}
	c.Flags().BoolVar(&mfa, "mfa", false, "send X-MFA-Verified: true (required for editor-role deletes)")
	return c
}

func newPostsAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Run character-frequency analysis across all posts (synchronous)",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := clientFromFlags().Analyze()
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
}

func newPostsReanalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reanalyze",
		Short: "Enqueue an async reanalysis job (requires RabbitMQ enabled)",
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID, err := clientFromFlags().Reanalyze()
			if err != nil {
				return err
			}
			fmt.Println(ok("Reanalysis job queued: " + yellow(jobID)))
			return nil
		},
	}
}

func newPostsExportCmd() *cobra.Command {
	var format, output string
	c := &cobra.Command{
		Use:   "export",
		Short: "Export all posts as JSON or CSV",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := clientFromFlags().ExportPosts(format)
			if err != nil {
				return err
			}
			if output == "" {
				fmt.Println(string(data))
				return nil
			}
			if err := os.WriteFile(output, data, 0o644); err != nil {
				return err
			}
			fmt.Println(ok(fmt.Sprintf("Wrote %s bytes to %s", cyan(fmt.Sprintf("%d", len(data))), cyan(output))))
			return nil
		},
	}
	c.Flags().StringVar(&format, "format", "json", "json or csv")
	c.Flags().StringVar(&output, "output", "", "write to file instead of stdout")
	return c
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
