package easyserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"

	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// easyserver: framework
// it has all of other servers needed things eg: tools middleware etc...

// gameserver: players frame sync play etc..
// roomserver: players matchmaking group stats etc...
// relationserver: frend etc...
// chatserver: chat
// loginserver: login get token...
// gateway: http -> grpc
// notifyserver: tcp ---- client
type IServer interface {
	BeforeRun(*ServiceOpt)
	Run(*ServiceOpt, grpc.ServiceRegistrar)
	// Config()
	// Tracer()
}

// a node has some roles use -roles="" to specify
// eg: ./easy-gos -roles="hello;chat"

// use -config="" to specify config path
// config file is a json file

type Server struct {
	Opt    *ServiceOpt
	Server IServer
	DB     *sqlx.DB
	Viper  *viper.Viper

	Trancer opentracing.Tracer
	Closer  io.Closer
}

type EasyServer struct {
	Servers map[string]*Server // k:servicename
	Flags   IServerFlags
}

var Opts *Options

func (es *EasyServer) LoadFlagConfig(Flags IServerFlags) {

	viper.SetConfigName(path.Base(Flags.GetPath())) // name of config file (without extension)
	viper.SetConfigType(Flags.GetConfigType())      // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(path.Dir(Flags.GetPath()))  // path to look for the config file in

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}

	f, err := os.Open(Flags.GetPath())
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	Opts = &Options{}
	decoder := json.NewDecoder(f)

	if err := decoder.Decode(Opts); err != nil {
		log.Fatalln(err)
	}
	es.Flags = Flags
}

func (es *EasyServer) FindOption(name string) *ServiceOpt {
	for _, opt := range Opts.Services {
		if opt.Name == name {
			return opt
		}
	}
	return nil
}

func (es *EasyServer) BuildServer(name string, server IServer) *EasyServer {

	opt := es.FindOption(name)
	if opt == nil {
		log.Fatalf("cant find service:%s /n", name)
	}

	var db *sqlx.DB
	if opt.Database != "" {
		db = NewDB(opt.Database, opt.PoolConns)
	}
	if len(es.Servers) == 0 {
		es.Servers = make(map[string]*Server)
	}
	tracer, closer := InitJaeger(name)
	es.Servers[opt.Name] = &Server{
		Opt:     opt,
		Server:  server,
		Viper:   viper.GetViper(),
		DB:      db,
		Trancer: tracer,
		Closer:  closer,
	}

	return es
}

// globle interceptor for all service's rpc request
func (es *EasyServer) GlobleInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	resp, err := handler(ctx, req)
	fmt.Println("ctx:", ctx, "resp", resp)
	return resp, err
}

func (es *EasyServer) GrpcServe(opt *ServiceOpt) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", opt.ListenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// regist interceptor
	var gopts []grpc.ServerOption
	gopts = append(gopts, grpc.UnaryInterceptor(es.GlobleInterceptor))
	s := grpc.NewServer(gopts...)

	return s, lis
}

func (es *EasyServer) Run(s *Server) {

	s.Server.BeforeRun(s.Opt)
	g, lis := es.GrpcServe(s.Opt)

	s.Server.Run(s.Opt, g)
	log.Printf("%s service listening at %v", s.Opt.Name, lis.Addr())
	if err := g.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (es *EasyServer) Serve() {
	ctx := context.Background()
	roles := es.Flags.GetRoles()
	if roles[0] == "all" {
		for _, v := range es.Servers {
			go es.Run(v)
		}
	} else {
		for _, role := range roles {
			if s, ok := es.Servers[role]; ok {
				go es.Run(s)
			} else {
				log.Printf("%s not exist", role)
			}
		}
	}

	<-ctx.Done()
}
