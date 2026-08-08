package cli

import (
	"encoding/json"
	"fmt"
	"os"
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

func printPostsTable(posts []Post) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSER\tTITLE\tCREATED")
	for _, p := range posts {
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", p.ID, p.UserID, truncate(p.Title, 50), p.CreatedAt)
	}
	w.Flush()
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
				fmt.Println("No posts found.")
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
			fmt.Printf("Created post #%d\n", post.ID)
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
			fmt.Printf("Updated post #%d\n", post.ID)
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
			fmt.Printf("Deleted post #%s\n", args[0])
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
			fmt.Printf("Reanalysis job queued: %s\n", jobID)
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
			fmt.Printf("Wrote %d bytes to %s\n", len(data), output)
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
