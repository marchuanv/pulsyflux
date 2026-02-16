#include <napi.h>
#include <windows.h>

typedef int (*ClientNewFunc)(const char*, const char*);
typedef int (*ClientPublishFunc)(int, const char*, int);
typedef int (*ClientSubscribeFunc)(int);
typedef int (*SubscriptionReceiveFunc)(int, void**, int*);
typedef int (*SubscriptionCloseFunc)(int);
typedef int (*ServerNewFunc)(const char*);
typedef int (*ServerStartFunc)(int);
typedef const char* (*ServerAddrFunc)(int);
typedef int (*ServerStopFunc)(int);
typedef void (*FreePayloadFunc)(void*);

static HMODULE hLib = nullptr;
static ClientNewFunc ClientNew = nullptr;
static ClientPublishFunc ClientPublish = nullptr;
static ClientSubscribeFunc ClientSubscribe = nullptr;
static SubscriptionReceiveFunc SubscriptionReceive = nullptr;
static SubscriptionCloseFunc SubscriptionClose = nullptr;
static ServerNewFunc ServerNew = nullptr;
static ServerStartFunc ServerStart = nullptr;
static ServerAddrFunc ServerAddr = nullptr;
static ServerStopFunc ServerStop = nullptr;
static FreePayloadFunc FreePayload = nullptr;

class ServerWrapper : public Napi::ObjectWrap<ServerWrapper> {
public:
  static Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Server", {
      InstanceMethod("start", &ServerWrapper::Start),
      InstanceMethod("addr", &ServerWrapper::Addr),
      InstanceMethod("stop", &ServerWrapper::Stop)
    });
    
    Napi::FunctionReference* constructor = new Napi::FunctionReference();
    *constructor = Napi::Persistent(func);
    env.SetInstanceData(constructor);
    
    exports.Set("Server", func);
    return exports;
  }
  
  ServerWrapper(const Napi::CallbackInfo& info) : Napi::ObjectWrap<ServerWrapper>(info) {
    std::string address = info[0].As<Napi::String>().Utf8Value();
    id_ = ServerNew(address.c_str());
  }
  
  Napi::Value Start(const Napi::CallbackInfo& info) {
    ServerStart(id_);
    return info.Env().Undefined();
  }
  
  Napi::Value Addr(const Napi::CallbackInfo& info) {
    const char* addr = ServerAddr(id_);
    return Napi::String::New(info.Env(), addr);
  }
  
  Napi::Value Stop(const Napi::CallbackInfo& info) {
    ServerStop(id_);
    return info.Env().Undefined();
  }
  
private:
  int id_;
};

class ClientWrapper : public Napi::ObjectWrap<ClientWrapper> {
public:
  static Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Client", {
      InstanceMethod("publish", &ClientWrapper::Publish),
      InstanceMethod("subscribe", &ClientWrapper::Subscribe)
    });
    
    Napi::FunctionReference* constructor = new Napi::FunctionReference();
    *constructor = Napi::Persistent(func);
    
    exports.Set("Client", func);
    return exports;
  }
  
  ClientWrapper(const Napi::CallbackInfo& info) : Napi::ObjectWrap<ClientWrapper>(info) {
    std::string address = info[0].As<Napi::String>().Utf8Value();
    std::string channelID = info[1].As<Napi::String>().Utf8Value();
    id_ = ClientNew(address.c_str(), channelID.c_str());
  }
  
  Napi::Value Publish(const Napi::CallbackInfo& info) {
    Napi::Buffer<char> buffer = info[0].As<Napi::Buffer<char>>();
    ClientPublish(id_, buffer.Data(), buffer.Length());
    return info.Env().Undefined();
  }
  
  Napi::Value Subscribe(const Napi::CallbackInfo& info) {
    int subID = ClientSubscribe(id_);
    return SubscriptionWrapper::NewInstance(info.Env(), subID);
  }
  
private:
  int id_;
  friend class SubscriptionWrapper;
};

class SubscriptionWrapper : public Napi::ObjectWrap<SubscriptionWrapper> {
public:
  static Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Subscription", {
      InstanceMethod("receive", &SubscriptionWrapper::Receive),
      InstanceMethod("close", &SubscriptionWrapper::Close)
    });
    
    constructor = Napi::Persistent(func);
    constructor.SuppressDestruct();
    
    exports.Set("Subscription", func);
    return exports;
  }
  
  static Napi::Object NewInstance(Napi::Env env, int subID) {
    Napi::Object obj = constructor.New({});
    SubscriptionWrapper* wrapper = Napi::ObjectWrap<SubscriptionWrapper>::Unwrap(obj);
    wrapper->id_ = subID;
    return obj;
  }
  
  SubscriptionWrapper(const Napi::CallbackInfo& info) : Napi::ObjectWrap<SubscriptionWrapper>(info) {
    id_ = -1;
  }
  
  Napi::Value Receive(const Napi::CallbackInfo& info) {
    void* payload = nullptr;
    int payloadLen = 0;
    
    int result = SubscriptionReceive(id_, &payload, &payloadLen);
    
    if (result < 0) {
      return info.Env().Null();
    }
    
    Napi::Buffer<char> buffer = Napi::Buffer<char>::Copy(info.Env(), (char*)payload, payloadLen);
    FreePayload(payload);
    
    return buffer;
  }
  
  Napi::Value Close(const Napi::CallbackInfo& info) {
    SubscriptionClose(id_);
    return info.Env().Undefined();
  }
  
private:
  int id_;
  static Napi::FunctionReference constructor;
};

Napi::FunctionReference SubscriptionWrapper::constructor;

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  hLib = LoadLibraryA("broker_lib.dll");
  if (!hLib) {
    Napi::Error::New(env, "Failed to load broker_lib.dll").ThrowAsJavaScriptException();
    return exports;
  }
  
  ClientNew = (ClientNewFunc)GetProcAddress(hLib, "ClientNew");
  ClientPublish = (ClientPublishFunc)GetProcAddress(hLib, "ClientPublish");
  ClientSubscribe = (ClientSubscribeFunc)GetProcAddress(hLib, "ClientSubscribe");
  SubscriptionReceive = (SubscriptionReceiveFunc)GetProcAddress(hLib, "SubscriptionReceive");
  SubscriptionClose = (SubscriptionCloseFunc)GetProcAddress(hLib, "SubscriptionClose");
  ServerNew = (ServerNewFunc)GetProcAddress(hLib, "ServerNew");
  ServerStart = (ServerStartFunc)GetProcAddress(hLib, "ServerStart");
  ServerAddr = (ServerAddrFunc)GetProcAddress(hLib, "ServerAddr");
  ServerStop = (ServerStopFunc)GetProcAddress(hLib, "ServerStop");
  FreePayload = (FreePayloadFunc)GetProcAddress(hLib, "FreePayload");
  
  ServerWrapper::Init(env, exports);
  ClientWrapper::Init(env, exports);
  SubscriptionWrapper::Init(env, exports);
  
  return exports;
}

NODE_API_MODULE(broker_addon, Init)
