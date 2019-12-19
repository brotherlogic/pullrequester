package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/goserver/utils"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbgh "github.com/brotherlogic/githubcard/proto"
	pbg "github.com/brotherlogic/goserver/proto"
	pb "github.com/brotherlogic/pullrequester/proto"
)

type github interface {
	getPullRequest(ctx context.Context, req *pbgh.PullRequest) (*pbgh.PullResponse, error)
	closePullRequest(ctx context.Context, req *pbgh.CloseRequest) (*pbgh.CloseResponse, error)
}

type prodGithub struct {
	dial func(server string) (*grpc.ClientConn, error)
}

func (p *prodGithub) getPullRequest(ctx context.Context, req *pbgh.PullRequest) (*pbgh.PullResponse, error) {
	conn, err := p.dial("githubcard")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbgh.NewGithubClient(conn)
	return client.GetPullRequest(ctx, req)
}

func (p *prodGithub) closePullRequest(ctx context.Context, req *pbgh.CloseRequest) (*pbgh.CloseResponse, error) {
	conn, err := p.dial("githubcard")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbgh.NewGithubClient(conn)
	return client.ClosePullRequest(ctx, req)
}

const (
	// KEY - where we store sale info
	KEY = "/github.com/brotherlogic/pullrequester/config"
)

//Server main server type
type Server struct {
	*goserver.GoServer
	config *pb.Config
	github github
}

func (s *Server) cleanTracking(ctx context.Context) error {
	for i, pr := range s.config.Tracking {
		elems := strings.Split(pr.Url, "/")
		if len(elems) > 7 {
			val, _ := strconv.Atoi(elems[7])
			prs, err := s.github.getPullRequest(ctx, &pbgh.PullRequest{Job: elems[5], PullNumber: int32(val)})

			if err != nil {
				return err
			}

			if !prs.IsOpen {
				s.config.Tracking = append(s.config.Tracking[:i], s.config.Tracking[i+1:]...)
				return s.cleanTracking(ctx)
			}
		} else {
			s.config.Tracking = append(s.config.Tracking[:i], s.config.Tracking[i+1:]...)
			return s.cleanTracking(ctx)
		}
	}

	return nil
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
		config:   &pb.Config{},
	}
	s.github = &prodGithub{dial: s.DialMaster}
	return s
}

func (s *Server) save(ctx context.Context) {
	s.KSclient.Save(ctx, KEY, s.config)
}

func (s *Server) load(ctx context.Context) error {
	config := &pb.Config{}
	data, _, err := s.KSclient.Read(ctx, KEY, config)

	if err != nil {
		return err
	}

	s.config = data.(*pb.Config)
	return nil
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	pb.RegisterPullRequesterServiceServer(server, s)
}

// ReportHealth alerts if we're not healthy
func (s *Server) ReportHealth() bool {
	return true
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.save(ctx)
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	if master {
		err := s.load(ctx)
		return err
	}

	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{
		&pbg.State{Key: "last_run", TimeValue: s.config.LastRun},
		&pbg.State{Key: "tracking", Text: fmt.Sprintf("%v", s.config.Tracking)},
	}
}

func main() {
	var quiet = flag.Bool("quiet", false, "Show all output")
	var init = flag.Bool("init", false, "Prep server")
	flag.Parse()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	server := Init()
	server.GoServer.KSclient = *keystoreclient.GetClient(server.DialMaster)
	server.PrepServer()
	server.Register = server
	err := server.RegisterServerV2("pullrequester", false, false)
	if err != nil {
		return
	}

	server.RegisterRepeatingTask(server.cleanTracking, "clean_tracking", time.Minute*5)

	if *init {
		ctx, cancel := utils.BuildContext("pullrequester", "pullrequester")
		defer cancel()
		server.config.LastRun = time.Now().Unix()
		server.save(ctx)
		return
	}

	fmt.Printf("%v", server.Serve())
}
