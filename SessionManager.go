package sessionmanager

import (
	"sync"
	"fmt"
	"io"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"time"
)

var(
	providers = make(map[string]Provider)
)

type Manager struct {
	cookieName string 	//private cookiename
	lock sync.Mutex		//protects session
	provider Provider
	maxLifeTime int64
}

type Provider interface {
	SessionInit(sid string) (Session, error)	//Implement initialization of session and returns new one if succeeded
	SessionRead(sid string) (Session, error)	//return session by id, creates one if not existant
	SessionDestroy(sid string) error			//deletes corresponding session
	SessionGC(maxLifeTime int64)				//deletes expired session variables according to maxLifeTime
}

type Session interface {
	Set(key, value interface{}) error	//set session value
	Get(key interface{}) interface{}	//get session value
	Delete(key interface{}) error		//delete session value
	SessionID() string					//get current sessionId
}

func NewManager(providerName, cookieName string, maxLifeTime int64) (*Manager, error){
	provider, ok := providers[providerName]
	if !ok {
		return nil, fmt.Errorf("session: unkown provide %q (forgotten import?)", providerName)
	}
	return &Manager{provider: provider, cookieName: cookieName, maxLifeTime: maxLifeTime}, nil
}

//Register makes a session provider available with provided name
//If a Register is called twice with the same name or if the driver is nil,
//it panics
func Register(name string, provider Provider){
	if provider == nil{
		panic("session: Register Provider is nil")
	}
	if _, dup := providers[name]; dup{
		panic("session: Register called twice for provider "+ name)
	}
	providers[name] = provider
}

//Create unique session IDs
func (manager *Manager) sessionId() string{
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

//Session GC
func (manager *Manager) GC(){
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.provider.SessionGC(manager.maxLifeTime)
	time.AfterFunc(time.Duration(manager.maxLifeTime), func() {
		manager.GC()
	})
}

//Reset sessions on, for example, logouts
func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == ""{
		return
	} else {
		manager.lock.Lock()
		defer manager.lock.Unlock()
		manager.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()
		cookie := http.Cookie{Name:manager.cookieName, Path:"/", HttpOnly:true, Expires:expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

//Checking existance of any sessions related to the current user
//creating a new session if none found
func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session Session){
	manager.lock.Lock()
	defer manager.lock.Unlock()
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == ""{
		sid := manager.sessionId()
		session, _ =manager.provider.SessionInit(sid)
		cookie := http.Cookie{Name: manager.cookieName, Value: url.QueryEscape(sid), Path:"/", HttpOnly:true, MaxAge:int(manager.maxLifeTime)}
		http.SetCookie(w, &cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = manager.provider.SessionRead(sid)
	}
	return //returns session!
}