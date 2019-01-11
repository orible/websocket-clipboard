String.prototype.trim = function (In_int_max, In_suffix) {
    if (this.length > 0)
        return this.substring(0, this.length > In_int_max ? In_int_max : this.length) + (((this.length > In_int_max) && In_suffix) ? In_suffix : "");
    return this;
}

var ModuleNotification = (function () {
    if (window.chrome){
        navigator.serviceWorker.register('sw.js');
    }

    function requestDesktopNotificationPermission() {
        if (Notification && Notification.permission !== 'granted') {
            //Notification.requestPermission(function (permission) {
           //     if (!('permission' in Notification)) {
            //        Notification.permission = permission;
           //     }
           // });
           Notification.requestPermission().then(
               function(permission) { 
                if (!('permission' in Notification)) {
                    Notification.permission = permission;
                }
            });
        }
    }
    function tryNotification(title, body, tag) {
        if (Notification.permission === "granted") {
            sendDesktopNotification(title, body);
            return
        }
        console.log("Permission not granted\n")
    }
    function sendDesktopNotification(title, body) {
        if (window.chrome) {
            navigator.serviceWorker.ready.then(function(registration) {
                registration.showNotification(
                    title, {
                        body: body,
                        icon: "asset/icon/icons8-download-from-cloud-50-pen.png",
                        vibrate: [200, 100, 200, 100, 200, 100, 200],
                        tag: "clipboard",
                    }
                );
            });
            return 0;
        }
        /*let notification = new window.Notification("Title", {
            //icon: "",
            //body: text,
            //tag: "soManyNotification"
        });
        notification.onshow = function() {
            console.log('Notification shown');
        };
        notification.onclick = function () {
            parent.focus();
            window.focus(); //just in case, older browsers
            this.close();
        };
        setTimeout(notification.close.bind(notification), 5000);*/
    }
    return {
        RequestPermission: function(){
            requestDesktopNotificationPermission();
        },
        SendEx: function (tile, body, tag, type) {
            body = body.trim(30, "...")
        },
        Send: function(title, body){
            //requestDesktopNotificationPermission();
            body = body.trim(100, "...")
            tryNotification(title, body);
        }
    }
}) ();