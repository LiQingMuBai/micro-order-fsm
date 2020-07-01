package main

import (
	"fmt"
	"time"

	"github.com/LiQingMuBai/micro-order-fsm/microservice/basic"
	"github.com/LiQingMuBai/micro-order-fsm/microservice/basic/common"
	"github.com/LiQingMuBai/micro-order-fsm/microservice/basic/config"
	tracer "github.com/LiQingMuBai/micro-order-fsm/microservice/plugins/tracer/jaeger"
	"github.com/LiQingMuBai/micro-order-fsm/microservice/user-srv/handler"
	"github.com/LiQingMuBai/micro-order-fsm/microservice/user-srv/model"
	s "github.com/LiQingMuBai/micro-order-fsm/microservice/user-srv/proto/user"
	"github.com/micro/cli"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/etcd"
	"github.com/micro/go-micro/util/log"
	"github.com/micro/go-plugins/config/source/grpc"
	openTrace "github.com/micro/go-plugins/wrapper/trace/opentracing"
	"github.com/opentracing/opentracing-go"
	"os"
)

var (
	appName = "user_srv"
	cfg     = &userCfg{}
)

type userCfg struct {
	common.AppCfg
}

func main() {
	// 初始化配置、数据库等信息
	initCfg()

	// 使用 Etcd 注册
	micReg := etcd.NewRegistry(registryOptions)

	traceAddr := os.Getenv("MICRO_BOOK_TRACER_ADDR")
	t, io, err := tracer.NewTracer(cfg.Name, traceAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)
	// 新建服务
	service := micro.NewService(
		micro.Name("mu.micro.book.srv.user"),
		micro.RegisterTTL(time.Second*15),
		micro.RegisterInterval(time.Second*10),
		micro.Registry(micReg),
		micro.Version("latest"),
		micro.WrapHandler(openTrace.NewHandlerWrapper(opentracing.GlobalTracer())),
	)

	// 服务初始化
	service.Init(
		micro.Action(func(c *cli.Context) {
			// 初始化模型层
			model.Init()
			// 初始化handler
			handler.Init()
		}),
	)

	// 注册服务
	s.RegisterUserHandler(service.Server(), new(handler.Service))

	// 启动服务
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

func registryOptions(ops *registry.Options) {
	etcdCfg := &common.Etcd{}
	err := config.C().App("etcd", etcdCfg)
	if err != nil {
		panic(err)
	}

	ops.Addrs = []string{fmt.Sprintf("%s:%d", etcdCfg.Host, etcdCfg.Port)}
}

func initCfg() {
	configAddr := os.Getenv("MICRO_BOOK_CONFIG_GRPC_ADDR")
	source := grpc.NewSource(
		grpc.WithAddress(configAddr),
		grpc.WithPath("micro"),
	)

	basic.Init(config.WithSource(source))

	err := config.C().App(appName, cfg)
	if err != nil {
		panic(err)
	}

	log.Logf("[initCfg] 配置，cfg：%v", cfg)

	return
}
