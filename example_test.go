package textselector_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/ipfs/go-cid"
	mipld "github.com/ipfs/go-ipld-format"
	mdag "github.com/ipfs/go-merkledag"
	mdagtest "github.com/ipfs/go-merkledag/test"

	// .../Raw must be imported to init() Raw-codec support
	// Dag-PB is additionally configured runtime
	dagpb "github.com/ipld/go-codec-dagpb"
	_ "github.com/ipld/go-ipld-prime/codec/raw"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/ipld/go-ipld-prime/traversal/selector"

	textselector "github.com/ipld/go-ipld-selector-text-lite"
)

func Example_selectorFromPath() {

	//
	// we put together a fixture datastore, and also return its root
	ds, rootCid := fixtureDagService()

	//
	// Selector spec from a path, hopefully within that rootCid
	// The 001 is deliberate: making sure index navigation still works
	selectorSpec, err := textselector.SelectorSpecFromPath("Links/1/Hash/Links/001/Hash", nil)
	if err != nil {
		log.Fatal(err)
	}

	//
	// this is how we R/O interact with the fixture DS
	linkSystem := cidlink.DefaultLinkSystem()
	linkSystem.StorageReadOpener = func(lctx ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		if cl, isCid := lnk.(cidlink.Link); !isCid {
			return nil, fmt.Errorf("unexpected link type %#v", lnk)
		} else {
			node, err := ds.Get(lctx.Ctx, cl.Cid)
			if err != nil {
				return nil, err
			}
			return bytes.NewBuffer(node.RawData()), nil
		}
	}

	//
	// this is what allows us to understand dagpb
	nodePrototypeChooser := dagpb.AddSupportToChooser(
		func(ipld.Link, ipld.LinkContext) (ipld.NodePrototype, error) {
			return basicnode.Prototype.Any, nil
		},
	)

	//
	// compile our selector from spec
	parsedSelector, err := selector.ParseSelector(selectorSpec.Node())
	if err != nil {
		log.Fatal(err)
	}

	//
	// Not sure why this indirection exists TBH...
	// Also trapping it in a struct ( twice ) is ugh
	ctx := context.TODO()
	linkContext := ipld.LinkContext{Ctx: ctx}

	//
	// this is how we pull the root node out of the DS
	startNodePrototype, err := nodePrototypeChooser(cidlink.Link{Cid: rootCid}, linkContext)
	if err != nil {
		log.Fatal(err)
	}
	startNode, err := linkSystem.Load(
		linkContext,
		cidlink.Link{Cid: rootCid},
		startNodePrototype,
	)
	if err != nil {
		log.Fatal(err)
	}

	//
	// this is executing the selector over the entire DS
	err = traversal.Progress{
		Cfg: &traversal.Config{
			Ctx:                            ctx,
			LinkSystem:                     linkSystem,
			LinkTargetNodePrototypeChooser: nodePrototypeChooser,
		},
	}.WalkAdv(
		startNode,
		parsedSelector,
		func(p traversal.Progress, n ipld.Node, r traversal.VisitReason) error {
			if r == traversal.VisitReason_SelectionMatch {
				content, _ := n.AsBytes()
				fmt.Printf("%s\n", content)
			}
			return nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// WantedNode
}

func fixtureDagService() (mipld.DAGService, cid.Cid) {
	ds := mdagtest.Mock()

	correct := mdag.NewRawNode([]byte("WantedNode"))
	incorrect := mdag.NewRawNode([]byte("UnwantedNode"))

	bothIncorrect := &mdag.ProtoNode{}
	bothIncorrect.AddNodeLink("a", incorrect)
	bothIncorrect.AddNodeLink("b", incorrect)

	bIsCorrect := &mdag.ProtoNode{}
	bIsCorrect.AddNodeLink("b", correct)
	bIsCorrect.AddNodeLink("a", incorrect)

	root := &mdag.ProtoNode{}
	root.AddNodeLink("", bothIncorrect)
	root.AddNodeLink("", bIsCorrect)

	ds.AddMany(context.Background(), []mipld.Node{
		correct, incorrect, bothIncorrect, bIsCorrect, root,
	})

	return ds, root.Cid()
}
