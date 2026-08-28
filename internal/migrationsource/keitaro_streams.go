package migrationsource

import (
	"fmt"
	"strings"
)

type keitaroStreamJSON struct {
	Name          string                  `json:"name"`
	Paths         []keitaroStreamPathJSON `json:"paths"`
	UnmappedNodes []string                `json:"unmapped_nodes"`
}

type keitaroStreamPathJSON struct {
	Weight int32                `json:"weight"`
	Lander keitaroFlowAssetJSON `json:"lander"`
	Offer  keitaroFlowAssetJSON `json:"offer"`
}

type keitaroFlowAssetJSON struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func parseKeitaroCampaignFlow(row keitaroCampaign, campaignIndex int) (*NormalizedFlow, []Warning) {
	if len(row.Streams) == 0 {
		return nil, nil
	}
	stream := row.Streams[0]
	var warnings []Warning
	ref := keitaroCampaignRef(row, campaignIndex)
	if len(row.Streams) > 1 {
		warnings = append(warnings, Warning{
			Slug:        "multiple_streams_truncated",
			Message:     fmt.Sprintf("imported first stream only (%d total)", len(row.Streams)),
			CampaignRef: ref,
		})
	}
	flow := &NormalizedFlow{
		Name:          strings.TrimSpace(stream.Name),
		UnmappedNodes: append([]string(nil), stream.UnmappedNodes...),
	}
	if flow.Name == "" {
		flow.Name = "imported-flow"
	}
	for i, path := range stream.Paths {
		landerName := strings.TrimSpace(path.Lander.Name)
		landerURL := strings.TrimSpace(path.Lander.URL)
		offerName := strings.TrimSpace(path.Offer.Name)
		offerURL := strings.TrimSpace(path.Offer.URL)
		if landerName == "" || landerURL == "" || offerName == "" || offerURL == "" {
			warnings = append(warnings, Warning{
				Slug:        "stream_path_incomplete",
				Message:     fmt.Sprintf("stream path index %d missing lander or offer fields", i),
				CampaignRef: ref,
			})
			continue
		}
		weight := path.Weight
		if weight <= 0 {
			weight = 100
		}
		landerRef := strings.TrimSpace(path.Lander.Ref)
		if landerRef == "" {
			landerRef = fmt.Sprintf("lander-%d", i+1)
		}
		offerRef := strings.TrimSpace(path.Offer.Ref)
		if offerRef == "" {
			offerRef = fmt.Sprintf("offer-%d", i+1)
		}
		flow.Paths = append(flow.Paths, NormalizedFlowPath{
			Weight: weight,
			Lander: NormalizedFlowAsset{Ref: landerRef, Name: landerName, URL: landerURL},
			Offer:  NormalizedFlowAsset{Ref: offerRef, Name: offerName, URL: offerURL},
		})
	}
	for _, node := range stream.UnmappedNodes {
		node = strings.TrimSpace(node)
		if node == "" {
			continue
		}
		warnings = append(warnings, Warning{
			Slug:        "stream_node_unmapped",
			Message:     "stream node not imported: " + node,
			CampaignRef: ref,
		})
	}
	if len(flow.Paths) == 0 {
		return nil, warnings
	}
	return flow, warnings
}

func keitaroCampaignRef(row keitaroCampaign, index int) string {
	ref := fmt.Sprintf("keitaro:%d", row.ID)
	if row.ID == 0 {
		ref = fmt.Sprintf("keitaro:%d", index+1)
	}
	return ref
}

func mapNormalizedFlow(flow *NormalizedFlow) *MappedFlow {
	if flow == nil || len(flow.Paths) == 0 {
		return nil
	}
	out := &MappedFlow{Name: flow.Name}
	for _, path := range flow.Paths {
		out.Paths = append(out.Paths, MappedFlowPath{
			Weight:     path.Weight,
			LanderRef:  path.Lander.Ref,
			LanderName: path.Lander.Name,
			LanderURL:  path.Lander.URL,
			OfferRef:   path.Offer.Ref,
			OfferName:  path.Offer.Name,
			OfferURL:   path.Offer.URL,
		})
	}
	return out
}
