#include <napi.h>
#include <windows.h>

typedef int (*NewServerFunc)(const char*);
typedef int (*StartFunc)(int);
typedef const char* (*AddrFunc)(int);
typedef int (*StopFunc)(int);
typedef int (*NewClientFunc)(const char*, const char*);
typedef int (*PublishFunc)(int, const char*, int);
typedef int (*SubscribeFunc)(int, void**, int*);
typedef void (*FreePayloadFunc)(void*);
typedef void (*CleanupFunc)();

static HMODULE hLib = nullptr;
static NewServerFunc NewServer = nullptr;
static StartFunc Start = nullptr;
static AddrFunc Addr = nullptr;
static StopFunc Stop = nullptr;
static NewClientFunc NewClient = nullptr;
static PublishFunc Publish = nullptr;
static SubscribeFunc Subscribe = nullptr;
static FreePayloadFunc FreePayload = nullptr;
static CleanupFunc Cleanup = nullptr;

class Server : public Napi::ObjectWrap<Server> {
public:
  static Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Server", {
      InstanceMethod("start", &Server::StartMethod),
      InstanceMethod("addr", &Server::AddrMethod),
      InstanceMethod("stop", &Server::StopMethod)
    });
    
    Napi::FunctionReference* constructor = new Napi::FunctionReference();
    *constructor = Napi::Persistent(func);
    env.SetInstanceData(constructor);
    
    exports.Set("Server", func);
    return exports;
  }
  
  Server(const Napi::CallbackInfo& info) : Napi::ObjectWrap<Server>(info) {
    std::string address = info[0].As<Napi::String>().Utf8Value();
    id_ = NewServer(address.c_str());
  }
  
  Napi::Value StartMethod(const Napi::CallbackInfo& info) {
    Start(id_);
    return info.Env().Undefined();
  }
  
  Napi::Value AddrMethod(const Napi::CallbackInfo& info) {
    const char* addr = Addr(id_);
    return Napi::String::New(info.Env(), addr);
  }
  
  Napi::Value StopMethod(const Napi::CallbackInfo& info) {
    Stop(id_);
    return info.Env().Undefined();
  }
  
private:
  int id_;
};

class Client : public Napi::ObjectWrap<Client> {
public:
  static Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Client", {
      InstanceMethod("publish", &Client::PublishMethod),
      InstanceMethod("subscribe", &Client::SubscribeMethod),
      InstanceMethod("subscribeAsync", &Client::SubscribeAsyncMethod)
    });
    
    Napi::FunctionReference* constructor = new Napi::FunctionReference();
    *constructor = Napi::Persistent(func);
    
    exports.Set("Client", func);
    return exports;
  }
  
  Client(const Napi::CallbackInfo& info) : Napi::ObjectWrap<Client>(info) {
    std::string address = info[0].As<Napi::String>().Utf8Value();
    std::string channelID = info[1].As<Napi::String>().Utf8Value();
    id_ = NewClient(address.c_str(), channelID.c_str());
  }
  
  Napi::Value PublishMethod(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int result = -1;
    
    if (info[0].IsString()) {
      std::string str = info[0].As<Napi::String>().Utf8Value();
      result = Publish(id_, (char*)str.c_str(), str.length());
    } else if (info[0].IsBuffer()) {
      Napi::Buffer<char> buffer = info[0].As<Napi::Buffer<char>>();
      result = Publish(id_, buffer.Data(), buffer.Length());
    } else {
      Napi::TypeError::New(env, "Expected string or buffer").ThrowAsJavaScriptException();
      return env.Undefined();
    }
    
    if (result < 0) {
      Napi::Error::New(env, "Publish failed").ThrowAsJavaScriptException();
      return env.Undefined();
    }
    
    return env.Undefined();
  }
  
  Napi::Value SubscribeMethod(const Napi::CallbackInfo& info) {
    void* payload = nullptr;
    int payloadLen = 0;
    
    int result = Subscribe(id_, &payload, &payloadLen);
    
    if (result < 0) {
      return info.Env().Null();
    }
    
    Napi::Buffer<char> buffer = Napi::Buffer<char>::Copy(info.Env(), (char*)payload, payloadLen);
    FreePayload(payload);
    
    return buffer;
  }

  Napi::Value SubscribeAsyncMethod(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    auto deferred = Napi::Promise::Deferred::New(env);
    
    // Check immediately for a message
    void* payload = nullptr;
    int payloadLen = 0;
    int result = Subscribe(id_, &payload, &payloadLen);
    
    if (result >= 0) {
      // Message available immediately
      Napi::Buffer<char> buffer = Napi::Buffer<char>::Copy(env, (char*)payload, payloadLen);
      FreePayload(payload);
      deferred.Resolve(buffer);
    } else {
      // No message, resolve with null
      deferred.Resolve(env.Null());
    }
    
    return deferred.Promise();
  }
  
private:
  int id_;
};

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  hLib = LoadLibraryA("broker_lib.dll");
  if (!hLib) {
    Napi::Error::New(env, "Failed to load broker_lib.dll").ThrowAsJavaScriptException();
    return exports;
  }
  
  NewServer = (NewServerFunc)GetProcAddress(hLib, "ServerNew");
  Start = (StartFunc)GetProcAddress(hLib, "ServerStart");
  Addr = (AddrFunc)GetProcAddress(hLib, "ServerAddr");
  Stop = (StopFunc)GetProcAddress(hLib, "ServerStop");
  NewClient = (NewClientFunc)GetProcAddress(hLib, "NewClient");
  Publish = (PublishFunc)GetProcAddress(hLib, "Publish");
  Subscribe = (SubscribeFunc)GetProcAddress(hLib, "Subscribe");
  FreePayload = (FreePayloadFunc)GetProcAddress(hLib, "FreePayload");
  Cleanup = (CleanupFunc)GetProcAddress(hLib, "Cleanup");
  
  env.AddCleanupHook([]() {
    if (Cleanup) Cleanup();
    if (hLib) {
      FreeLibrary(hLib);
      hLib = nullptr;
    }
  });
  
  Server::Init(env, exports);
  Client::Init(env, exports);
  
  exports.Set("cleanup", Napi::Function::New(env, [](const Napi::CallbackInfo& info) {
    if (Cleanup) Cleanup();
    return info.Env().Undefined();
  }));
  
  return exports;
}

NODE_API_MODULE(broker_addon, Init)
