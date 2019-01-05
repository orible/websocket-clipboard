var test;
const NetLibCore = (function () {
    const CLIENT_VERSION = "0.0.1";
    var connection = new c_connection();
    var messages = 0;
    function c_connection(flags) {
        var socket;
        var serverClosing = false;
        var actions = Array();
        var uuid = 0;

        const BAD = 0;

        const SERVER_PUSH = 10;
        const SERVER_CONNECT_OK = 11;
        const SERVER_CLOSING = 12;
        const SERVER_RESPONSE_OK = 13;
        const SERVER_RESPONSE_BAD = 14;
        const CLIENT_CONNECT = 1;
        const CLIENT_DISCONNECT = 2;
        const CLIENT_LIST = 3;
        const CLIENT_PAIR_ROLL = 4;
        const CLIENT_PAIR_CONNECT = 5;
        const CLIENT_PUSH_CLIPBOARD = 6;

        function _SocketSend(e) {
            let message = JSON.stringify(e);
            socket.send(message);
            return 0;
        }
        function createCallbackEx(type, cb, timeout, flag, id) {
            console.log("createCallbackEx -> timeout", timeout);
            actions.push({ Type: type, Id: id, cb: cb, time: Date.now(), timeout: timeout, flag: flag })
        }
        function createCallback(type, cb, timeout, flag) {
            uuid++;
            actions.push({ Type: type, Id: uuid, cb: cb, time: Date.now(), timeout: timeout, flag: flag })
            return uuid;
        }
        function _SendAction(Type, Object, cb) {
            return _SocketSend({
                Type: Type,
                Time: + new Date(),
                Callback: createCallback(Type, cb, 1000 * 10, 0x0),
                Transport: Object,
            });
        }
        function _Send(Type, Object) {
            return _SocketSend({
                Type: Type,
                Time: + new Date(),
                Callback: -1,
                Transport: Object,
            });
        }
        function _onOpen() {
            if (serverClosing)
                return
            _Send(CLIENT_CONNECT, {});
        }
        function _onMessage(evt) {
            var json = JSON.parse(evt.data);
            messages++;
            for (var i = 0; i < json.Data.length; i++) {
                msg = json.Data[i];
                let time = Date.now() - msg.Time;
                if (time < 0)
                    alert("bad clock!");
                console.log("Time: %f\n", time / 1000);
                let data = msg.Transport;
                switch (msg.Type) {
                    case SERVER_CONNECT_OK:
                        console.log("Server -> \n\t%s\n\t%s\n\t%s\n",
                            data.Uptime,
                            data.Users,
                            data.Version,
                        );
                        break
                    case SERVER_CLOSING:
                        serverClosing = true
                        break
                    default:
                        if (msg.Type != SERVER_RESPONSE_BAD || msg.Type != SERVER_RESPONSE_OK) {
                            break
                        }
                        let codeName = ((msg.Type == SERVER_RESPONSE_OK) ? "OK": "BAD")
                        for (let i = 0; i < actions.length; i++) {
                            let a = actions[i];
                            if (a.Id == msg.Callback) {
                                let time = Date.now() - a.Time;
                                console.log("Server -> Response [%s] [%d] [%d]\n", codeName, a.Type, time);
                                let ret = a.cb({ type: a.Type, id: a.Id, data: data });
                                if (ret != -1) {
                                    actions.splice(i, 1);
                                }
                                break
                            }
                        }
                        break
                }
            }
        }
        function _onClose() {
            if (serverClosing) {

            } else {
                alert("server closed without close flag");
            }
        }
        function _Tick() {
            for (let i = 0; i < actions.length; i++) {
                if (actions[i].time + actions[i].timeout < Date.now()) {
                    let a = actions[i];
                    console.log("Action -> timed out\n");
                    let ret = a.cb(
                        {
                            type: -1,
                            id: a.Id,
                            data: null
                        })
                    actions.splice(i, 1);
                }
            }
            setTimeout(_Tick, 1000);
        }
        return {
            CreateExpectantCallback: function (type, id, cbf, timeout) {
                return createCallbackEx(type, cbf, timeout, 0x1, id);
            },
            ConnectPair: function (key, cb) {
                _SendAction(CLIENT_PAIR_CONNECT, {
                    Key: key,
                }, cb);
            },
            NewPair: function (cb) {
                _SendAction(CLIENT_PAIR_ROLL, null, cb);
            },
            Send: function (type, object) {
                _Send(type, object);
            },
            Connect: function () {
                socket = new WebSocket("ws://" + document.location.host + "/ws");
                socket.addEventListener('message', _onMessage.bind(this));
                socket.addEventListener('close', _onClose.bind(this));
                socket.addEventListener('open', _onOpen.bind(this));
                _Tick();
            }
        }
    }
    window.onload = function () {
        connection.Connect();
        test = connection;


        btnRollPair.onclick = function (e) {
            connection.NewPair(function (e) {
                console.log(e);
                if (e.type < 0) {
                    console.log("net -> failed to create connection pair\n");
                    return;
                }
                viewRollPairBox.value = e.data.Key;
                connection.CreateExpectantCallback(0, e.id, function (e) {
                    console.log("client connected -> ", e);
                    return true;
                }, e.data.Timeout);
                return false;
            });
        }
        btnConnectPair.onclick = function (e) {
            connection.ConnectPair(parseInt(viewConnectPairBox.value), function (e) {
                console.log(e);
                if (e.type < 0) {
                    console.log("net -> failed to connect\n");
                    return;
                }
                viewConnectPairBox.value = e.data.Key;
                return true;
            });
        }
    }
})();

(function () {
    var element = document.querySelectorAll("tbody#lol tr");
    var str = "const(";
    element.forEach(element => {
        x = element.childNodes;
        str += x[0].innerHTML + " = " + x[1].innerHTML + "\n";
    });
    str += ")"
    copy(str)
})();

test.NewPair(function (v) { console.log(v); });