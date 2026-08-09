package cli

import "fmt"

// bannerPAW spells "PAW" in block letters. Printed once, at REPL
// startup only — one-shot CLI commands never show it, so scripting
// against the CLI output stays unaffected.
var bannerPAW = []string{
	`  _____   __          __ `,
	` |  __ \ /\ \        / / `,
	` | |__) /  \ \  /\  / /  `,
	` |  ___/ /\ \ \/  \/ /   `,
	` | |  / ____ \  /\  /    `,
	` |_| /_/    \_\/  \/     `,
}

// bannerPalette rotates a blue -> magenta -> cyan gradient down the
// banner, echoing the blue/purple/teal accent colors used across the
// rest of the project's branding (see index.html).
var bannerPalette = []func(string) string{blue, blue, magenta, magenta, cyan, cyan}

func printBanner() {
	for i, line := range bannerPAW {
		fmt.Println(bannerPalette[i%len(bannerPalette)](bold(line)))
	}
	fmt.Println()
}
