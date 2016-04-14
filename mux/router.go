package mux

import (
	"github.com/suboat/sorm/log"
	"github.com/suboat/go-response/session"
	"github.com/suboat/go-response"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
)

// 虚拟websocket路由
type wsHandler struct {
	HandlerDefault *response.LogicHandler
	HandlerGet     *response.LogicHandler
	HandlerPost    *response.LogicHandler
	HandlerPut     *response.LogicHandler
	HandlerDelete  *response.LogicHandler
	HandlerOptions *response.LogicHandler
}

func (w *wsHandler) Bind(methodLis ...string) (err error) {
	if len(methodLis) == 0 {
		return
	}
	for _, method := range methodLis {
		switch method {
		case response.RequestCrudRead:
			w.HandlerGet = w.HandlerDefault
			break
		case response.RequestCrudCreate:
			w.HandlerPost = w.HandlerDefault
			break
		case response.RequestCrudUpdate:
			w.HandlerPut = w.HandlerDefault
			break
		case response.RequestCrudDelete:
			w.HandlerDelete = w.HandlerDefault
			break
		case response.RequestCrudOptions:
			w.HandlerOptions = w.HandlerDefault
			break
		default:
			// error
			log.Error("method: ", method)
			return
		}
	}
	w.HandlerDefault = nil
	return
}

func (w *wsHandler) Handle(req *response.RequestMode) (res *response.Response) {
	var h *response.LogicHandler
	//
	switch req.Method {
	case response.RequestCrudRead:
		h = w.HandlerGet
		break
	case response.RequestCrudCreate:
		h = w.HandlerPost
		break
	case response.RequestCrudUpdate:
		h = w.HandlerPut
		break
	case response.RequestCrudDelete:
		h = w.HandlerDelete
		break
	default:
		// unsupport error
		log.Error("wsHandler unsupport error")
		res = response.NewResponse(req)
		res.Error = response.ErrRequestSupport
		return
	}
	// default
	if h == nil {
		h = w.HandlerDefault
		log.Debug(req.Method, w.HandlerDefault)
		res = response.NewResponse(req)
		res.Error = response.ErrRequestSupport
		return
	}
	res = (*h)(req)
	return
}

type wsRouterMap map[string]*wsHandler

func (m *wsRouterMap) Methods(url string, methods ...string) (err error) {
	if h, ok := (*m)[url]; ok {
		(*h).Bind(methods...)
	}
	return
}

type WsRouter struct {
	Map    *wsRouterMap
	Hub    *response.HubWs
	Prefix string
}
type WsRoute struct {
	Map *wsRouterMap
	Url string
}

// http路由以及websocket虚拟路由
type Router struct {
	// http router
	*mux.Router
	// socket router
	*WsRouter
}

// inherit from mux
type Route struct {
	// http router
	*mux.Route
	// socket router
	*WsRoute
}
type RouteMatch struct {
	*mux.RouteMatch
}

// 处理websocket
func (r *Router) ListenAndServeWs(path string, opt interface{}) (err error) {
	go r.WsRouter.Hub.Run()
	r.Router.HandleFunc(path, r.ServeWebSocket)
	return
}
func (r *Router) ServeWebSocket(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(rw, "Method not allowed", 405)
		return
	}

	var (
		uid string
		se  *session.Session
		ws  *websocket.Conn
		c   *response.ConnWs
		err error
	)

	if ws, err = response.WsUpgrader.Upgrade(rw, req, nil); err != nil {
		log.Error(err.Error())
		return
	}

	// init uid
	if se, err = session.HttpSessionUid(rw, req); err != nil {
		http.Error(rw, err.Error(), 405)
		return
	} else {
		uid = se.Uid
	}

	// conn
	c = &response.ConnWs{
		Uid:      uid,
		Send:     make(chan []byte, 256),
		SendText: make(chan string),
		Ws:       ws,
		Hub:      r.Hub,
	}

	// handler
	c.Handler = func(req *response.RequestMode) (res *response.Response) {
		// TODO: 将实际URL转为定义URL
		log.Debug("ws recive url: ", req.Url)

		if h, ok := (*r.WsRouter.Map)[req.Url]; ok {
			res = h.Handle(req)
		} else {
			res = response.NewResponse(req)
			res.Error = response.ErrRequestSupport
			for url, _ := range *r.WsRouter.Map {
				log.Debug("ws: url map ", url)
			}
		}
		return
	}

	//response.HubWsSet.Register <- c
	r.Hub.Register <- c
	go c.WritePump()
	c.ReadPump()
}

func (r *WsRouter) Handle(path string, handler response.LogicHandler) {
	if h, ok := (*r.Map)[path]; ok {
		h.HandlerDefault = &handler
	} else {
		(*r.Map)[path] = &wsHandler{
			HandlerDefault: &handler,
		}
	}
	//println("handle:", path, r.Map, len(*r.Map))
	return
}

// rewrite: Router
func (r *Router) Handle(path string, handler response.RestHandler) (rt *Route) {
	rt = new(Route)
	rt.Route = r.Router.Handle(path, handler)
	//println("hhhh", path, r.Router, handler)
	//
	rt.WsRoute = newWsRoute(nil)
	rt.WsRoute.Map = r.WsRouter.Map
	//
	rt.WsRoute.Url = r.WsRouter.Prefix + path
	r.WsRouter.Handle(rt.Url, handler.ServeLogic)
	return
}
func (r *Router) HandleFunc(path string,
	f func(http.ResponseWriter, *http.Request)) (rt *Route) {
	rt = new(Route)
	rt.Route = r.Router.HandleFunc(path, f)
	return
}
func (r *Router) PathPrefix(tpl string) (rt *Route) {
	rt = new(Route)
	rt.Route = r.Router.PathPrefix(tpl)
	//
	rt.WsRoute = newWsRoute(nil)
	rt.WsRoute.Map = r.WsRouter.Map
	// TODO: bug fix
	r.WsRouter.Prefix = tpl
	rt.WsRoute.Url = tpl
	return
}

// rewrite: Route
func (r *Route) Subrouter() (rt *Router) {
	rt = NewRouter()
	rt.Router = r.Route.Subrouter()
	rt.WsRouter.Map = r.WsRoute.Map
	rt.WsRouter.Prefix = r.WsRoute.Url
	return
}

func (r *Route) Methods(methods ...string) *Route {
	r.Route = r.Route.Methods(methods...)
	r.WsRoute.Map.Methods(r.Url, methods...)
	return r
}

// new one
func newWsRouter(src *WsRouter) (r *WsRouter) {
	r = new(WsRouter)
	if src != nil {
		r.Map = src.Map
	} else {
		r.Map = newWsRouterMap()
	}
	r.Hub = response.NewHubWs(nil)
	return
}
func newWsRoute(src *WsRoute) (r *WsRoute) {
	r = new(WsRoute)
	if src != nil {
		r.Map = src.Map
	} else {
		r.Map = newWsRouterMap()
	}
	return
}

func newWsRouterMap() *wsRouterMap {
	m := make(map[string]*wsHandler)
	wm := wsRouterMap(m)
	return &wm
}

func NewRouter() (r *Router) {
	r = new(Router)
	r.Router = mux.NewRouter()
	r.WsRouter = newWsRouter(nil)
	return
}

func Vars(r *http.Request) map[string]string {
	return mux.Vars(r)
}
