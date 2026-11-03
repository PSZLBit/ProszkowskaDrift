package pszlbit.drift.controller

/**
 * Transport abstraction used by the controller client.
 *
 * The production implementation wraps an OkHttp 3 WebSocket; tests use a fake
 * transport that plays the server side. Keeping the boundary transport-shaped
 * lets the connection state machine be unit tested without a socket.
 */
interface Transport {
    /** Registers the listener and opens the connection. */
    fun open(listener: TransportListener)

    /** Sends an encoded protocol envelope. */
    fun send(envelope: Envelope)

    /** Closes the connection. */
    fun close()
}

interface TransportListener {
    /** Called when the WebSocket has been opened. */
    fun onOpened()

    /** Called for every inbound protocol envelope. */
    fun onEnvelope(envelope: Envelope)

    /** Called when the connection was closed or failed. */
    fun onClosed(code: Int, reason: String)
}
