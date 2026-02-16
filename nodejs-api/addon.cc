#include <napi.h>
#include <windows.h>
#include <thread>
#include <chrono>
#include <atomic>

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

class MessageWorker : public Napi::AsyncWorker {
public:
  MessageWorker(Napi::Function& callback, int clientId, Napi::FunctionReference* callbackRef)
    : Napi::AsyncWorker(callback), clientId_(clientId), callbackRef_(callbackRef), payload_(nullptr), payloadLen_(0) {}

  ~MessageWorker() {
    if (payload_) {
      FreePayload(payload_);
    }
  }

  void Execute() override {
    // Poll for messages with small delays
    for (int i = 0; i < 100; i++) {
      int result = Subscribe(clientId_, &payload_, &payloadLen_);
      if (result == 0 && payload_) {
        return; // Got a message
      }
      std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
    // No message found after polling
    payload_ = nullptr;
    payloadLen_ = 0;
  }

  void OnOK() override {
    Napi::HandleScope scope(Env());
    if (payload_) {
      Napi::Buffer<char> buffer = Napi::Buffer<char>::Copy(Env(), (char*)payload_, payloadLen_);
      Callback().Call({buffer});
    }
    // Schedule next check if callback still exists
    if (callbackRef_ && !callbackRef_->IsEmpty()) {
      Napi::Function cb = callbackRef_->Value();
      MessageWorker* worker = new MessageWorker(cb, clientId_, callbackRef_);
      worker->Queue();
    }
  }

private:
  int clientId_;
  Napi::FunctionReference* callbackRef_;
  void* payload_;
  int payloadLen_;
};

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
      InstanceMethod("onMessage", &Client::OnMessageMethod)
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

  Napi::Value OnMessageMethod(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    
    if (info.Length() < 1 || !info[0].IsFunction()) {
      Napi::TypeError::New(env, "Expected callback function").ThrowAsJavaScriptException();
      return env.Undefined();
    }
    
    callback_ = Napi::Persistent(info[0].As<Napi::Function>());
    ScheduleNextCheck();
    
    return env.Undefined();
  }
  
  void ScheduleNextCheck() {
    if (!callback_.IsEmpty()) {
      Napi::Function cb = callback_.Value();
      MessageWorker* worker = new MessageWorker(cb, id_, &callback_);
      worker->Queue();
    }
  }
  
private:
  int id_;
  Napi::FunctionReference callback_;
};

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  // Try loading from same directory as addon
  hLib = LoadLibraryA("broker_lib.dll");
  if (!hLib) {
    // Try with full path relative to addon location
    char path[MAX_PATH];
    HMODULE hModule;
    GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, (LPCSTR)&Init, &hModule);
    GetModuleFileNameA(hModule, path, MAX_PATH);
    std::string dllPath = path;
    size_t pos = dllPath.find_last_of("\\");
    if (pos != std::string::npos) {
      dllPath = dllPath.substr(0, pos + 1) + "broker_lib.dll";
      hLib = LoadLibraryA(dllPath.c_str());
    }
  }
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
