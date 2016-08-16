function SocketConnector (connector) {
    return function (done) {
        var socket = connector();
        socket.on('connect', function () {
            socket.removeAllListeners();
            done(null, socket);
        }).on('error', done);
    };
}

module.exports = SocketConnector;
