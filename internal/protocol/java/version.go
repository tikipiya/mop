package java

// DefaultProtocolVersion is the Java Edition protocol used in the handshake.
// It is kept in one replaceable constant because Status support is generally
// version-tolerant but not guaranteed to be so for every server implementation.
const DefaultProtocolVersion int32 = 776 // Minecraft Java Edition 26.2
