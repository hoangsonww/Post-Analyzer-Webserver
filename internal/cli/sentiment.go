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
			paint := sentimentColor(label)
			fmt.Printf("sentiment: %s\n", paint(bold(label)))
			for k, v := range probs {
				c := sentimentColor(k)
				bar := strings.Repeat("█", int(v*20))
				fmt.Printf("  %-10s %s %s\n", k, c(bar), dim(fmt.Sprintf("%.1f%%", v*100)))
			}
			return nil
		},
	}
}

// sentimentColor maps a label (positive/negative/neutral) to the color
// used for both the label itself and its probability bar — green/red
// carry their usual connotation; neutral gets yellow rather than a
// third arbitrary hue.
func sentimentColor(label string) func(string) string {
	switch label {
	case "positive":
		return green
	case "negative":
		return red
	default:
		return yellow
	}
}
