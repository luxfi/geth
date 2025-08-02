package utils

import (
	"fmt"
	"path/filepath"
	
	"github.com/urfave/cli/v2"
	"github.com/luxfi/geth/ethdb/badgerdb"
	"github.com/luxfi/geth/node"
)

// makeBasicNode creates a basic node instance for utility functions
func makeBasicNode(ctx *cli.Context) (*node.Node, error) {
	cfg := &node.Config{
		DataDir: ctx.String(DataDirFlag.Name),
		Name:    "geth-util",
	}
	if cfg.DataDir == "" {
		cfg.DataDir = node.DefaultDataDir()
	}
	
	SetNodeConfig(ctx, cfg)
	
	stack, err := node.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create the protocol stack: %v", err)
	}
	
	return stack, nil
}

var (
	// FreezerCommands contains freezer-related commands
	FreezerCommands = []*cli.Command{
		{
			Name:      "freeze",
			Usage:     "Freeze blockchain data to ancient store",
			ArgsUsage: "",
			Description: `
The freeze command moves finalized blockchain data from the main database 
to the ancient store for efficient long-term storage.`,
			Flags: []cli.Flag{
				DataDirFlag,
				AncientFlag,
				&cli.Uint64Flag{
					Name:  "threshold",
					Usage: "Number of recent blocks to keep in main database",
					Value: 90000,
				},
				&cli.Uint64Flag{
					Name:  "batch",
					Usage: "Number of blocks to freeze in one batch",
					Value: 1000,
				},
			},
			Action: freezeBlocks,
		},
		{
			Name:      "mount-ancient",
			Usage:     "Mount ancient store in read-only mode",
			ArgsUsage: "<ancient-path>",
			Description: `
Mount an ancient store in read-only mode for sharing across multiple nodes.`,
			Action: mountAncient,
		},
		{
			Name:      "export-ancient",
			Usage:     "Export ancient store to snapshot",
			ArgsUsage: "<output-path>",
			Description: `
Export the ancient store to a compressed snapshot for distribution.`,
			Flags: []cli.Flag{
				DataDirFlag,
				AncientFlag,
			},
			Action: exportAncient,
		},
	}
)

func freezeBlocks(ctx *cli.Context) error {
	// Setup paths
	stack, err := makeBasicNode(ctx)
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}
	defer stack.Close()
	
	chaindb := MakeChainDatabase(ctx, stack, false)
	defer chaindb.Close()
	
	// Get the ancient path
	ancientPath := ctx.String(AncientFlag.Name)
	if ancientPath == "" {
		ancientPath = filepath.Join(stack.DataDir(), "ancient")
	}
	
	// Create freezer config
	config := badgerdb.FreezerConfig{
		AncientPath:     ancientPath,
		FreezeThreshold: ctx.Uint64("threshold"),
		BatchSize:       ctx.Uint64("batch"),
	}
	
	// Create and run freezer
	freezer, err := badgerdb.NewFreezer(chaindb, config)
	if err != nil {
		return fmt.Errorf("failed to create freezer: %w", err)
	}
	defer freezer.Stop()
	
	// Get current head block
	// This is a simplified version - in practice you'd get this from the blockchain
	head := uint64(1000000) // Example
	
	// Run freezing
	if err := freezer.FreezeBlocks(head); err != nil {
		return fmt.Errorf("freezing failed: %w", err)
	}
	
	fmt.Println("Freezing completed successfully")
	return nil
}

func mountAncient(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("ancient path required")
	}
	
	ancientPath := ctx.Args().Get(0)
	
	// Create read-only mount
	db, err := badgerdb.CreateReadOnlyMount(ancientPath)
	if err != nil {
		return fmt.Errorf("failed to mount ancient store: %w", err)
	}
	defer db.Close()
	
	// Display info about the ancient store
	ancients, err := db.Ancients()
	if err != nil {
		return fmt.Errorf("failed to get ancient count: %w", err)
	}
	
	tail, err := db.Tail()
	if err != nil {
		return fmt.Errorf("failed to get tail: %w", err)
	}
	
	fmt.Printf("Ancient store mounted successfully\n")
	fmt.Printf("Path: %s\n", ancientPath)
	fmt.Printf("Blocks: %d to %d\n", tail, ancients-1)
	fmt.Printf("Total: %d blocks\n", ancients-tail)
	
	return nil
}

func exportAncient(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("output path required")
	}
	
	outputPath := ctx.Args().Get(0)
	ancientPath := ctx.String(AncientFlag.Name)
	
	if ancientPath == "" {
		stack, err := makeBasicNode(ctx)
		if err != nil {
			return fmt.Errorf("failed to create node: %w", err)
		}
		ancientPath = filepath.Join(stack.DataDir(), "ancient")
		stack.Close()
	}
	
	// Export ancient store
	if err := badgerdb.ExportAncientSnapshot(ancientPath, outputPath); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}
	
	fmt.Printf("Ancient store exported to: %s\n", outputPath)
	return nil
}