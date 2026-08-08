package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSentimentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sentiment <text...>",
		Short: "Classify the sentiment of text via Triton (requires TRITON_ENABLED on the gateway)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			label, probs, err := clientFromFlags().Sentiment(strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Printf("sentiment: %s\n", label)
			for k, v := range probs {
				fmt.Printf("  %-10s %.4f\n", k, v)
			}
			return nil
		},
	}
}
