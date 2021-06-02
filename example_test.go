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

	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	_ "github.com/ipld/go-ipld-prime/codec/raw"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal"

	textselector "github.com/ipld/go-ipld-selector-text-lite"
)

const FixturePath = "Links/1/Hash/Links/1/Hash"

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

func ExampleSelectorFromPath() {

	ctx := context.TODO()

	// we put together a fixture datastore, and also return its root
	ds, rootCid := fixtureDagService()

	// we put our selector together
	parsedSelector, err := textselector.SelectorFromPath(FixturePath)
	if err != nil {
		log.Fatal(err)
	}

	// not sure what this is for TBH...
	linkContext := ipld.LinkContext{Ctx: ctx}

	// this is what allows us to understand dagpb
	nodePrototypeChooser := dagpb.AddSupportToChooser(
		func(ipld.Link, ipld.LinkContext) (ipld.NodePrototype, error) {
			return basicnode.Prototype.Any, nil
		},
	)

	// this is how we interact with the fixture DS
	linkSystem := cidlink.DefaultLinkSystem()
	linkSystem.StorageReadOpener = func(_ ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		if cl, isCid := lnk.(cidlink.Link); !isCid {
			return nil, fmt.Errorf("unexpected link type %#v", lnk)
		} else {
			node, err := ds.Get(context.TODO(), cl.Cid)
			if err != nil {
				return nil, err
			}
			return bytes.NewBuffer(node.RawData()), nil
		}
	}

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
		func(p traversal.Progress, n ipld.Node, _ traversal.VisitReason) error {

			if n.Kind() == ipld.Kind_Bytes {
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
