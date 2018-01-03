package memory

import (
	"container/list"
	"github.com/ArgonautDevelopments/SessionManager"
	"sync"
	"time"
)

//SESSION

type Provider struct {
	lock sync.Mutex						//lock
	sessions map[string]*list.Element 	//save in memory
	list *list.List						//gc
}

var prov = &Provider{list: list.New()}

type SessionStore struct{
	sid string							// unique session id
	timeAccessed time.Time				//last access time
	value map[interface{}]interface{}	//session value stored inside
}

func (st *SessionStore) Set(key, value interface{}) error {
	st.value[key] = value
	prov.SessionUpdate(st.sid)
	return nil
}

func (st *SessionStore) Get(key interface{}) interface{} {
	prov.SessionUpdate(st.sid)
	if v, ok := st.value[key]; ok {
		return v
	} else{
		return nil
	}
}

func (st *SessionStore) SessionID() string {
	return st.sid
}

func (st *SessionStore) Delete(key interface{}) error {
	delete(st.value, key)
	prov.SessionUpdate(st.sid)
	return nil
}

//PROVIDER

func (pder *Provider) SessionInit(sid string) (sessionmanager.Session, error){
	prov.lock.Lock()
	defer prov.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	newSess := &SessionStore{sid: sid, timeAccessed: time.Now(), value: v}
	element := prov.list.PushBack(newSess)
	prov.sessions[sid] = element
	return newSess, nil
}

func (prov *Provider) SessionRead(sid string) (sessionmanager.Session, error){
	if element, ok := prov.sessions[sid]; ok {
		return element.Value.(*SessionStore), nil
	} else {
		sess, err := prov.SessionInit(sid)
		return sess, err
	}
}

func (prov *Provider) SessionDestroy(sid string) error {
	if element, ok := prov.sessions[sid]; ok {
		delete(prov.sessions, sid)
		prov.list.Remove(element)
	}
	return nil
}

func (prov *Provider) SessionGC(maxLifeTime int64){
	prov.lock.Lock()
	defer prov.lock.Unlock()

	for {
		element := prov.list.Back()
		if element == nil {
			break
		}

		if (element.Value.(*SessionStore).timeAccessed.Unix() + maxLifeTime) < time.Now().Unix() {
			prov.list.Remove(element)
			delete(prov.sessions, element.Value.(*SessionStore).sid)
		} else {
			break
		}
	}
}

func (prov *Provider) SessionUpdate(sid string) error{
	prov.lock.Lock()
	defer prov.lock.Unlock()
	if element, ok := prov.sessions[sid]; ok {
		element.Value.(*SessionStore).timeAccessed = time.Now()
		prov.list.MoveToFront(element)
		return nil
	}
	return nil
}

func init(){
	prov.sessions = make(map[string]*list.Element, 0)
	sessionmanager.Register("memory", prov)
}

