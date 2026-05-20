package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/proto"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
)

// runAnnouncement is the entry point for the `pth announcement ...`
// group introduced in M5.C phase 10. Subcommands:
//
//	announcement create --playtest-slug --send-to --subject --message
//	announcement list   --playtest-slug
//
// PRD §5.4 "Bulk announcements" / STATUS_M5.md C10.
func runAnnouncement(ctx context.Context, stdout, stderr io.Writer, g *Globals, args []string, factory playtestClientFactory) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "announcement: usage: pth announcement {create|list} ...")
		return exitLocalError
	}
	action, rest := args[0], args[1:]
	switch action {
	case "create":
		return runAnnouncementCreate(ctx, stdout, stderr, g, rest, factory)
	case actionList:
		return runAnnouncementList(ctx, stdout, stderr, g, rest, factory)
	default:
		fmt.Fprintf(stderr, "announcement: unknown action %q\n", action)
		return exitLocalError
	}
}

func runAnnouncementCreate(ctx context.Context, stdout, stderr io.Writer, g *Globals, args []string, factory playtestClientFactory) int {
	fs := flag.NewFlagSet("announcement create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	playtestID := fs.String("playtest-id", "", "playtest UUID (required)")
	sendTo := fs.String("send-to", "APPROVED_ONLY", "recipient filter: ALL|APPROVED_ONLY|PENDING_ONLY")
	subject := fs.String("subject", "", "broadcast subject (required, 1-200 chars)")
	msg := fs.String("message", "", "broadcast message body (required, 1-4000 chars)")
	dryRun := fs.Bool("dry-run", false, "print the request JSON and exit without dialling")
	if err := fs.Parse(args); err != nil {
		return exitLocalError
	}
	if *playtestID == "" {
		fmt.Fprintln(stderr, "announcement create: --playtest-id is required")
		return exitLocalError
	}
	if *subject == "" {
		fmt.Fprintln(stderr, "announcement create: --subject is required")
		return exitLocalError
	}
	if *msg == "" {
		fmt.Fprintln(stderr, "announcement create: --message is required")
		return exitLocalError
	}
	filter, ok := parseSendToFilter(*sendTo)
	if !ok {
		fmt.Fprintf(stderr, "announcement create: --send-to must be ALL|APPROVED_ONLY|PENDING_ONLY, got %q\n", *sendTo)
		return exitLocalError
	}
	if !g.requireNamespace(stderr, "announcement create") {
		return exitLocalError
	}
	req := &pb.CreateAnnouncementRequest{
		Namespace:    g.Namespace,
		PlaytestId:   *playtestID,
		SendToFilter: filter,
		Subject:      *subject,
		Message:      *msg,
	}
	return invokePlaytest(ctx, stdout, stderr, g, factory, "CreateAnnouncement", req, *dryRun,
		func(c pb.PlaytesthubServiceClient, cctx context.Context) (proto.Message, error) {
			return c.CreateAnnouncement(cctx, req)
		})
}

func runAnnouncementList(ctx context.Context, stdout, stderr io.Writer, g *Globals, args []string, factory playtestClientFactory) int {
	fs := flag.NewFlagSet("announcement list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	playtestID := fs.String("playtest-id", "", "playtest UUID (required)")
	dryRun := fs.Bool("dry-run", false, "print the request JSON and exit without dialling")
	if err := fs.Parse(args); err != nil {
		return exitLocalError
	}
	if *playtestID == "" {
		fmt.Fprintln(stderr, "announcement list: --playtest-id is required")
		return exitLocalError
	}
	if !g.requireNamespace(stderr, "announcement list") {
		return exitLocalError
	}
	req := &pb.ListAnnouncementsRequest{
		Namespace:  g.Namespace,
		PlaytestId: *playtestID,
	}
	return invokePlaytest(ctx, stdout, stderr, g, factory, "ListAnnouncements", req, *dryRun,
		func(c pb.PlaytesthubServiceClient, cctx context.Context) (proto.Message, error) {
			return c.ListAnnouncements(cctx, req)
		})
}

func parseSendToFilter(raw string) (pb.AnnouncementSendToFilter, bool) {
	switch strings.ToUpper(raw) {
	case "ALL":
		return pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_ALL, true
	case "APPROVED_ONLY":
		return pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_APPROVED_ONLY, true
	case "PENDING_ONLY":
		return pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_PENDING_ONLY, true
	}
	return pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_UNSPECIFIED, false
}
