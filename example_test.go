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

	"github.com/ipld/go-ipld-prime"
	dagpb "github.com/ipld/go-ipld-prime-proto"
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

	parsedSelector, err := textselector.SelectorFromPath(FixturePath)
	if err != nil {
		log.Fatal(err)
	}

	ds, rootCid := fixtureDagService()

	linkBlockLoader := func(lnk ipld.Link, _ ipld.LinkContext) (io.Reader, error) {
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

	nilLinkCtx := ipld.LinkContext{}
	nodeStyleChooser := dagpb.AddDagPBSupportToChooser(
		func(ipld.Link, ipld.LinkContext) (ipld.NodeStyle, error) {
			return basicnode.Style.Any, nil
		},
	)

	nodeStyle, err := nodeStyleChooser(cidlink.Link{Cid: rootCid}, nilLinkCtx)
	if err != nil {
		log.Fatal(err)
	}

	startNodeBuilder := nodeStyle.NewBuilder()
	if err := (cidlink.Link{Cid: rootCid}).Load(
		context.TODO(),
		nilLinkCtx,
		startNodeBuilder,
		linkBlockLoader,
	); err != nil {
		log.Fatal(err)
	}
	startNode := startNodeBuilder.Build()

	err = traversal.Progress{
		Cfg: &traversal.Config{
			LinkLoader:                 linkBlockLoader,
			LinkTargetNodeStyleChooser: nodeStyleChooser,
		},
	}.WalkAdv(
		startNode,
		parsedSelector,
		func(p traversal.Progress, n ipld.Node, _ traversal.VisitReason) error {
			if rawNode, ok := n.(*dagpb.RawNode); ok {
				content, _ := rawNode.AsBytes()
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
