// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
)

type answerResult struct {
	Package     answers.Package `json:"package"`
	Freshness   string          `json:"freshness"`
	CacheHit    bool            `json:"cache_hit"`
	Diagnostics []string        `json:"diagnostics"`
	Usage       answers.Usage   `json:"usage"`
}
type answerContinuation struct{ Epoch, Request, Cursor string }
type answerWalk struct {
	node  string
	path  answers.Path
	depth int
	after string
}
type answerMeter struct {
	budget                                   answers.Budget
	usage                                    answers.Usage
	reserveRows, reserveBytes, reserveRounds int
}

func (m *answerMeter) room() (int, int, int) {
	return m.budget.Rows - m.usage.Rows - m.reserveRows, m.budget.SourceBytes - m.usage.SourceBytes - m.reserveBytes, m.budget.Rounds - m.usage.Rounds - m.reserveRounds
}
func (m *answerMeter) begin() bool {
	rows, bytes, rounds := m.room()
	if rows < 1 || bytes < 1 || rounds < 1 {
		return false
	}
	m.usage.Rounds++
	return true
}
func answerRequestDigest(r answers.Request) string { r.Cursor = ""; return answers.Digest(r) }
func answerDiagnostic(p *answers.Package, reason string) {
	if !slices.Contains(p.Diagnostics, reason) {
		p.Diagnostics = append(p.Diagnostics, reason)
	}
	if p.State != "cancelled" {
		p.State = "partial"
	}
}

func (a *App) answerGraph(ctx context.Context, p *answers.Package, m *answerMeter) (graph.Adapter, graph.Checkpoint) {
	if !m.begin() {
		answerDiagnostic(p, "retrieval_budget")
		return nil, graph.Checkpoint{}
	}
	status, err := a.Catalog.GraphStatus(ctx)
	if err != nil {
		p.Observation.GraphState = "unavailable"
		answerDiagnostic(p, "graph_unavailable")
		return nil, graph.Checkpoint{}
	}
	if !m.begin() {
		answerDiagnostic(p, "retrieval_budget")
		return nil, graph.Checkpoint{}
	}
	g, err := a.openGraph(ctx)
	if err != nil {
		p.Observation.GraphState = "unavailable"
		answerDiagnostic(p, "graph_unavailable")
		return nil, graph.Checkpoint{}
	}
	if !m.begin() {
		answerDiagnostic(p, "retrieval_budget")
		return nil, graph.Checkpoint{}
	}
	cp, err := g.ReadCheckpoint(ctx, p.WorkspaceID, a.Catalog.GraphTarget())
	if err != nil {
		p.Observation.GraphState = "unavailable"
		answerDiagnostic(p, "graph_unavailable")
		return nil, graph.Checkpoint{}
	}
	p.Observation.GraphGeneration = strconv.FormatInt(cp.Generation, 10)
	p.Observation.GraphSequence = strconv.FormatInt(cp.Sequence, 10)
	p.Observation.GraphRevision = strconv.FormatInt(cp.CatalogRevision, 10)
	if status["pending_events"] != 0 || status["checkpoint"] != cp.Sequence || status["generation"] != cp.Generation {
		p.Observation.GraphState = "lagging"
		answerDiagnostic(p, "graph_lag")
		return nil, cp
	}
	if cp.Sequence == 0 {
		p.Observation.GraphState = "missing"
		answerDiagnostic(p, "graph_projection_missing")
		return nil, cp
	}
	p.Observation.GraphState = "current"
	return g, cp
}

func (a *App) prepareAnswer(parent context.Context, request answers.Request) (out answerResult, err error) {
	r, err := answers.Normalize(request)
	if err != nil {
		return out, err
	}
	defer func() {
		if out.Package.State == "cancelled" && err == nil {
			err = contracts.Fail("cancelled")
		}
	}()
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, time.Duration(r.Budget.DeadlineMS)*time.Millisecond)
	defer cancel()
	defer func() {
		if ctx.Err() != nil {
			err = contracts.Fail("cancelled")
			out.Freshness = "unknown"
			if !slices.Contains(out.Diagnostics, "cancelled") {
				out.Diagnostics = append(out.Diagnostics, "cancelled")
			}
		}
	}()
	p := answers.Package{Kind: answerArtifactKind, Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, State: "complete", Request: r, Items: []answers.Item{}, Conflicts: []answers.Conflict{}, Exclusions: []answers.Exclusion{}, Diagnostics: []string{}, Observation: answers.Observation{GraphState: "unobserved"}}
	out.Freshness = "unknown"
	out.Diagnostics = []string{}
	if ctx.Err() != nil {
		p.State = "cancelled"
		p.Diagnostics = append(p.Diagnostics, "cancelled")
		out.Package = p
		return out, nil
	}
	if err = a.lockGraph(ctx); err != nil {
		p.State = "cancelled"
		p.Diagnostics = append(p.Diagnostics, "cancelled")
		out.Package = p
		return out, nil
	}
	defer a.graphMu.Unlock()
	m := answerMeter{budget: r.Budget}
	defer func() { out.Usage = m.usage; out.Usage.ElapsedMS = time.Since(started).Milliseconds() }()
	m.usage.Rounds++
	epoch, err := a.Catalog.AnswerEpoch(ctx, r.Filters.MediaIDs, max(1, r.Budget.Rows/3), max(1, r.Budget.SourceBytes/4))
	m.usage.Rows += epoch.Rows
	m.usage.SourceBytes += epoch.Bytes
	if err != nil {
		if ctx.Err() == nil {
			return out, err
		}
		p.State = "cancelled"
	} else if !epoch.Complete {
		answerDiagnostic(&p, "source_identity_budget")
	}
	p.Observation.SourceEpoch = epoch.Digest
	p.Observation.CatalogRevision = strconv.FormatInt(epoch.Revision, 10)
	if !epoch.Complete || ctx.Err() != nil {
		p.Usage = m.usage
		p.Usage.ElapsedMS = time.Since(started).Milliseconds()
		out.Package = p
		return out, nil
	}
	m.reserveRows = 2 * epoch.Rows
	m.reserveBytes = 2 * epoch.Bytes
	m.reserveRounds = 2
	var continuation answerContinuation
	if r.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(r.Cursor)
		if e != nil || contracts.DecodeExplore(raw, &continuation) != nil || continuation.Request != answerRequestDigest(r) {
			return out, contracts.Fail("invalid_request")
		}
		if continuation.Epoch != epoch.Digest {
			return out, contracts.Condition("conflict", "source_authority_changed")
		}
	}
	g, cp := a.answerGraph(ctx, &p, &m)
	graphEpoch := answers.Digest([]any{p.Observation.GraphState, cp, a.Workspace.Config.Profiles.Graph})
	p.Key = answers.CacheKey(r, p.WorkspaceID, epoch.Digest, graphEpoch)
	var pointer answerPointer
	var pointerRevision int64
	if m.begin() {
		raw, revision, e := a.Catalog.NamedSetting(ctx, "answer-context:"+p.Key)
		pointerRevision = revision
		if e != nil {
			return out, e
		}
		if len(raw) > 0 {
			if contracts.DecodeExplore(raw, &pointer) == nil && pointer.valid(p.WorkspaceID, p.Key) {
				if m.begin() {
					_, cacheBytes, _ := m.room()
					cached, n, e := a.readAnswerPackageBound(ctx, pointer.ID, pointer.Digest, cacheBytes)
					m.usage.SourceBytes += n
					if e == nil && cached.Key == p.Key && reflect.DeepEqual(cached.Request, r) && cached.Observation.SourceEpoch == epoch.Digest && answerCacheable(cached) {
						m.reserveRows = 0
						m.reserveBytes = 0
						m.reserveRounds = 0
						if m.begin() {
							rows, bytes, _ := m.room()
							after, e := a.Catalog.AnswerEpoch(ctx, r.Filters.MediaIDs, rows, bytes)
							m.usage.Rows += after.Rows
							m.usage.SourceBytes += after.Bytes
							if e == nil && after.Complete && after.Digest == epoch.Digest {
								out.Package = cached
								out.CacheHit = true
								out.Freshness = "current"
								return out, nil
							}
						}
						return out, contracts.Condition("conflict", "source_authority_changed")
					}
					out.Diagnostics = append(out.Diagnostics, "cache_content_rejected")
				}
			} else {
				out.Diagnostics = append(out.Diagnostics, "cache_index_rejected")
			}
		}
	}
	selected := map[string]int{}
	add := func(item answers.Item, path answers.Path, expanded bool) bool {
		if !answers.Matches(item, r, expanded) {
			return true
		}
		if i, ok := selected[item.ID]; ok {
			if len(p.Items[i].Paths) < 16 {
				p.Items[i].Paths = append(p.Items[i].Paths, path)
			}
			return true
		}
		if len(p.Items) >= r.Budget.Items {
			answerDiagnostic(&p, "evidence_item_budget")
			return false
		}
		item.Paths = append(item.Paths, path)
		raw, _ := json.Marshal(item)
		if m.usage.EvidenceBytes+len(raw) > r.Budget.EvidenceBytes {
			answerDiagnostic(&p, "evidence_byte_budget")
			p.Exclusions = append(p.Exclusions, answers.Exclusion{EvidenceID: item.ID, Reason: "evidence_byte_budget"})
			return false
		}
		m.usage.EvidenceBytes += len(raw)
		selected[item.ID] = len(p.Items)
		p.Items = append(p.Items, item)
		return true
	}
	zone := "UTC"
	if r.Filters.RecordingDates != nil && r.Filters.RecordingDates.Timezone != "" {
		zone = r.Filters.RecordingDates.Timezone
	}
	read := func(query catalog.AnswerPageRequest) (catalog.AnswerPage, bool) {
		if ctx.Err() != nil || !m.begin() {
			answerDiagnostic(&p, "retrieval_budget")
			return catalog.AnswerPage{}, false
		}
		rows, bytes, _ := m.room()
		query.Limit = min(query.Limit, rows)
		query.MaxBytes = bytes
		query.Zone = zone
		query.MediaIDs = r.Filters.MediaIDs
		page, e := a.Catalog.AnswerCandidates(ctx, query)
		m.usage.Rows += page.Rows
		m.usage.SourceBytes += page.Bytes
		if e != nil {
			answerDiagnostic(&p, "catalog_read_failed")
			return page, false
		}
		if page.LimitReason != "" {
			answerDiagnostic(&p, page.LimitReason)
		}
		return page, true
	}
	cursor := continuation.Cursor
	catalogDone := false
	for ctx.Err() == nil && len(p.Items) < r.Budget.Items {
		page, ok := read(catalog.AnswerPageRequest{Cursor: cursor, Limit: min(32, r.Budget.Items-len(p.Items))})
		if !ok {
			break
		}
		for _, item := range page.Items {
			add(item, answers.Path{Kind: "catalog-filter", Nodes: []string{item.ID}}, false)
		}
		cursor = page.Next
		if page.LimitReason != "" {
			break
		}
		if cursor == "" {
			catalogDone = true
			break
		}
	}
	if !catalogDone {
		answerDiagnostic(&p, "catalog_scan_incomplete")
		if cursor != "" {
			raw, _ := json.Marshal(answerContinuation{Epoch: epoch.Digest, Request: answerRequestDigest(r), Cursor: cursor})
			p.Next = base64.RawURLEncoding.EncodeToString(raw)
		}
	}
	seeds := slices.Clone(p.Items)
	for _, seed := range seeds {
		if *r.Budget.NeighborCues == 0 || len(p.Items) >= r.Budget.Items {
			break
		}
		ordinal, _ := answers.Decimal(seed.Reference.Ordinal, 0)
		lo, hi := max(int64(0), ordinal-int64(*r.Budget.NeighborCues)), ordinal+int64(*r.Budget.NeighborCues)
		page, ok := read(catalog.AnswerPageRequest{RecordingID: seed.Reference.RecordingID, FirstOrdinal: &lo, LastOrdinal: &hi, Limit: 2**r.Budget.NeighborCues + 1})
		if !ok {
			break
		}
		for _, item := range page.Items {
			add(item, answers.Path{Kind: "neighbor-cue", Nodes: []string{seed.ID, item.ID}}, true)
		}
	}
	queue := []answerWalk{}
	for _, seed := range seeds {
		queue = append(queue, answerWalk{node: seed.ID, path: answers.Path{Kind: "graph", Nodes: []string{seed.ID}}})
	}
	for _, id := range r.Filters.EntityIDs {
		queue = append(queue, answerWalk{node: id, path: answers.Path{Kind: "graph", Nodes: []string{id}}})
	}
	for _, id := range r.Filters.ConceptIDs {
		node := "concept:" + id
		queue = append(queue, answerWalk{node: node, path: answers.Path{Kind: "graph", Nodes: []string{node}}})
	}
	seenNodes := map[string]bool{}
	for g != nil && len(queue) > 0 && ctx.Err() == nil {
		walk := queue[0]
		queue = queue[1:]
		if walk.depth >= r.Traversal.MaxDepth || walk.after == "" && seenNodes[walk.node] {
			continue
		}
		seenNodes[walk.node] = true
		if m.usage.Edges >= r.Budget.Edges || len(p.Items) >= r.Budget.Items || !m.begin() {
			answerDiagnostic(&p, "graph_traversal_budget")
			break
		}
		rows, bytes, _ := m.room()
		if rows < 2 {
			answerDiagnostic(&p, "graph_traversal_budget")
			break
		}
		page, e := g.Neighborhood(ctx, graph.NeighborhoodRequest{Workspace: p.WorkspaceID, Target: a.Catalog.GraphTarget(), Checkpoint: cp, Node: walk.node, Direction: r.Traversal.Direction, Kinds: r.Traversal.RelationshipTypes, After: walk.after, Limit: min(rows-1, r.Budget.Edges-m.usage.Edges), MaxBytes: bytes})
		if e != nil {
			answerDiagnostic(&p, "graph_read_failed")
			break
		}
		m.usage.Rows += page.Rows
		if !page.NodePresent && page.LimitReason == "" {
			answerDiagnostic(&p, "graph_projection_missing")
		}
		m.usage.Edges += len(page.Edges)
		m.usage.SourceBytes += page.Bytes
		if page.LimitReason != "" {
			answerDiagnostic(&p, page.LimitReason)
			break
		}
		if page.Next != "" {
			next := walk
			next.after = page.Next
			queue = append(queue, next)
		}
		for _, edge := range page.Edges {
			node := edge.To
			if node == walk.node {
				node = edge.From
			}
			path := answers.Path{Kind: "graph", Nodes: append(slices.Clone(walk.path.Nodes), node), Edges: append(slices.Clone(walk.path.Edges), edge.ID)}
			parts := strings.SplitN(node, ":", 4)
			if len(parts) == 4 && parts[0] == "cue" && contracts.ValidID(parts[1]) {
				candidates, ok := read(catalog.AnswerPageRequest{RecordingID: parts[1], CueID: parts[3], Limit: 2})
				if !ok {
					break
				}
				if len(candidates.Items) != 1 || candidates.Items[0].Reference.DocumentDigest != parts[2] {
					p.Exclusions = append(p.Exclusions, answers.Exclusion{EvidenceID: node, Reason: "stale-or-missing-graph-cue"})
					answerDiagnostic(&p, "graph_authority_missing")
				} else {
					add(candidates.Items[0], path, true)
				}
			}
			if !seenNodes[node] {
				queue = append(queue, answerWalk{node: node, path: path, depth: walk.depth + 1})
			}
		}
	}
	if ctx.Err() != nil {
		p.State = "cancelled"
		p.Diagnostics = append(p.Diagnostics, "cancelled")
		p.Usage = m.usage
		p.Usage.ElapsedMS = time.Since(started).Milliseconds()
		out.Package = p
		return out, nil
	}
	m.reserveRows = epoch.Rows
	m.reserveBytes = epoch.Bytes
	m.reserveRounds = 1
	if !m.begin() {
		answerDiagnostic(&p, "freshness_unverified")
	} else {
		rows, bytes, _ := m.room()
		after, e := a.Catalog.AnswerEpoch(ctx, r.Filters.MediaIDs, rows, bytes)
		m.usage.Rows += after.Rows
		m.usage.SourceBytes += after.Bytes
		if e == nil && after.Complete && after.Digest == epoch.Digest {
			out.Freshness = "current"
		} else if e == nil && after.Complete {
			out.Freshness = "stale"
			answerDiagnostic(&p, "source_authority_changed")
		} else {
			answerDiagnostic(&p, "freshness_unverified")
		}
	}
	// Re-rank the retained set after byte pruning so counts, overlap groups and
	// conflict references never describe evidence absent from the package.
	for {
		if ctx.Err() != nil {
			p.State = "cancelled"
			out.Package = p
			return out, nil
		}
		var excluded []answers.Exclusion
		p.Items, p.Conflicts, excluded, p.IndependentFamilies = answers.Rank(r.Question, p.Items)
		m.usage.EvidenceBytes = 0
		kept := []answers.Item{}
		for _, item := range p.Items {
			raw, _ := json.Marshal(item)
			if m.usage.EvidenceBytes+len(raw) > r.Budget.EvidenceBytes {
				answerDiagnostic(&p, "evidence_byte_budget")
				p.Exclusions = append(p.Exclusions, answers.Exclusion{EvidenceID: item.ID, Reason: "ranked_evidence_byte_budget"})
			} else {
				kept = append(kept, item)
				m.usage.EvidenceBytes += len(raw)
			}
		}
		if len(kept) == len(p.Items) {
			p.Exclusions = append(p.Exclusions, excluded...)
			break
		}
		p.Items = kept
	}
	p.Context, err = answers.BuildContext(r, p.Items)
	if err != nil {
		return out, err
	}
	if p.Context.Omitted > 0 {
		answerDiagnostic(&p, "context_omissions")
	}
	if len(p.Items) == 0 && p.State == "complete" {
		p.State = "insufficient"
		p.Diagnostics = append(p.Diagnostics, "no_matching_evidence")
	}
	p.Usage = m.usage
	p.Usage.ElapsedMS = time.Since(started).Milliseconds()
	if out.Freshness != "current" {
		out.Package = p
		return out, nil
	}
	p.ID = contracts.ID()
	if err = a.saveAnswerPackage(ctx, p, pointerRevision); err != nil {
		if ctx.Err() != nil {
			p.State = "cancelled"
			p.ID = ""
			out.Package = p
			return out, nil
		}
		return out, err
	}
	// Persistence is separate from semantic source identity. Recheck it after
	// publication; a cache pointer cannot make a concurrent source edit current.
	m.reserveRows = 0
	m.reserveBytes = 0
	m.reserveRounds = 0
	if !m.begin() {
		out.Freshness = "unknown"
		out.Diagnostics = append(out.Diagnostics, "post_publication_freshness_unverified")
		out.Package = p
		return out, nil
	}
	rows, bytes, _ := m.room()
	post, e := a.Catalog.AnswerEpoch(ctx, r.Filters.MediaIDs, rows, bytes)
	m.usage.Rows += post.Rows
	m.usage.SourceBytes += post.Bytes
	if e != nil || !post.Complete {
		out.Freshness = "unknown"
		out.Diagnostics = append(out.Diagnostics, "post_publication_freshness_unverified")
	} else if post.Digest != epoch.Digest {
		out.Freshness = "stale"
		out.Diagnostics = append(out.Diagnostics, "source_authority_changed")
	}
	out.Package = p
	return out, nil
}

func answerCacheable(p answers.Package) bool {
	if p.ID == "" || p.Next != "" || p.State == "cancelled" || p.Observation.SourceEpoch == "" {
		return false
	}
	for _, d := range p.Diagnostics {
		if !slices.Contains([]string{"graph_unavailable", "graph_lag", "graph_projection_missing", "context_omissions", "no_matching_evidence"}, d) {
			return false
		}
	}
	return true
}
