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

	gip "github.com/ipld/go-ipld-prime"
	gippb "github.com/ipld/go-ipld-prime-proto"
	gipcidlink "github.com/ipld/go-ipld-prime/linking/cid"
	gipbasicnode "github.com/ipld/go-ipld-prime/node/basic"
	giptraversal "github.com/ipld/go-ipld-prime/traversal"

	textselector "github.com/ribasushi/go-ipld-selector-text-lite"
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

	linkBlockLoader := func(lnk gip.Link, _ gip.LinkContext) (io.Reader, error) {
		if cl, isCid := lnk.(gipcidlink.Link); !isCid {
			return nil, fmt.Errorf("unexpected link type %#v", lnk)
		} else {
			node, err := ds.Get(context.TODO(), cl.Cid)
			if err != nil {
				return nil, err
			}
			return bytes.NewBuffer(node.RawData()), nil
		}
	}

	gipNilCtx := gip.LinkContext{}
	gipNodeStyleChooser := gippb.AddDagPBSupportToChooser(
		func(gip.Link, gip.LinkContext) (gip.NodeStyle, error) {
			return gipbasicnode.Style.Any, nil
		},
	)

	gipNodeStyle, err := gipNodeStyleChooser(gipcidlink.Link{Cid: rootCid}, gipNilCtx)
	if err != nil {
		log.Fatal(err)
	}

	startNodeBuilder := gipNodeStyle.NewBuilder()
	if err := (gipcidlink.Link{Cid: rootCid}).Load(
		context.TODO(),
		gipNilCtx,
		startNodeBuilder,
		linkBlockLoader,
	); err != nil {
		log.Fatal(err)
	}
	startNode := startNodeBuilder.Build()

	err = giptraversal.Progress{
		Cfg: &giptraversal.Config{
			LinkLoader:                 linkBlockLoader,
			LinkTargetNodeStyleChooser: gipNodeStyleChooser,
		},
	}.WalkAdv(
		startNode,
		parsedSelector,
		func(p giptraversal.Progress, n gip.Node, _ giptraversal.VisitReason) error {
			if rawNode, ok := n.(*gippb.RawNode); ok {
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
