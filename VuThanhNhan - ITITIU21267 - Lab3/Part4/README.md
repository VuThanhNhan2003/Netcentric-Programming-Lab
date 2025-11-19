# Part 4: Hybrid Chat System

## Overview
This is a real-world hybrid chat application that uses both TCP and UDP protocols:
- **TCP (Port 9000)**: Reliable message delivery with message history
- **UDP (Port 9001)**: Real-time status updates (typing, online, away)

## Features Implemented

### Server Features
✅ **TCP Server** - Handles reliable message delivery
- Listens on port 9000
- Manages multiple client connections simultaneously
- Stores last 100 messages in circular buffer
- Broadcasts messages to all connected clients
- Handles client disconnections gracefully

✅ **UDP Server** - Handles real-time status updates
- Listens on port 9001
- Tracks user statuses (online, typing, away)
- Broadcasts status updates to all clients
- Sends acknowledgments back to clients

✅ **Message History**
- Stores up to 100 messages with timestamps
- Provides history on client request
- Automatic history on join (last 10 messages)

✅ **Thread-Safe Operations**
- Uses mutex locks for concurrent access
- Separate goroutines for TCP and UDP servers
- Safe client registration/deregistration

### Client Features
✅ **Dual Protocol Connection**
- Connects to both TCP and UDP servers
- Handles TCP messages and UDP status updates simultaneously

✅ **Interactive Commands**
- `/status <online|typing|away>` - Change your status
- `/history <count>` - Request message history
- `/quit` - Exit the chat
- Type any text to send a message

✅ **Automatic Features**
- Heartbeat every 5 seconds to maintain presence
- Automatic typing status when sending messages
- Real-time status notifications from other users

## How to Run

### Step 1: Start the Server
```powershell
# Navigate to Part4 directory
cd "c:\Users\VUTHANHHUNG\Desktop\Netcen Pro\VuThanhNhan - ITITIU21267 - Lab3\Part4"

# Run the server
go run hybrid_chat_server.go
```

Expected output:
```
=== Hybrid Chat Server ===
TCP Server listening on :9000
UDP Server listening on :9001
```

### Step 2: Start Client(s)
Open new terminal windows for each client:

```powershell
# In a new terminal
cd "c:\Users\VUTHANHHUNG\Desktop\Netcen Pro\VuThanhNhan - ITITIU21267 - Lab3\Part4"
go run hybrid_chat_client.go
```

### Step 3: Use the Chat

**Client 1 (Alice):**
```
Connected to chat server (TCP: 9000, UDP: 9001)
Enter username: Alice
*** Joined chat room ***

Commands:
  /status <online|typing|away> - Change your status
  /history <count> - Request message history
  /quit - Exit the chat
  Type any message to send to the chat room

> Hello everyone!
[10:30:15] Alice: Hello everyone!
```

**Client 2 (Bob):**
```
Connected to chat server (TCP: 9000, UDP: 9001)
Enter username: Bob
*** Joined chat room ***
[10:30:15] Alice: Hello everyone!

> Hi Alice!
Status Update: Bob is typing
[10:30:18] Bob: Hi Alice!
Status Update: Bob is online
```

## Message Protocol

### TCP Messages
- **Chat Message**: `MSG:<username>:<message>`
- **History Request**: `HISTORY:<count>`
- **Server Response**: `[HH:MM:SS] username: message`

### UDP Messages
- **Status Update**: `STATUS:<username>:<status>`
  - Status types: `online`, `typing`, `away`
- **Acknowledgment**: `ACK:<username>:<status>`

## Architecture

```
┌─────────────────────────────────────────────┐
│         Hybrid Chat Server                  │
├─────────────────────────────────────────────┤
│  TCP Server (9000)      UDP Server (9001)   │
│  • Messages             • Status Updates    │
│  • History              • Heartbeats        │
│  • Broadcasts           • Presence          │
└─────────────────────────────────────────────┘
         ↕                      ↕
┌─────────────────────────────────────────────┐
│         Client (Goroutines)                 │
├─────────────────────────────────────────────┤
│  • TCP Receiver      • UDP Receiver         │
│  • Message Sender    • Status Sender        │
│  • User Input        • Heartbeat Timer      │
└─────────────────────────────────────────────┘
```

## Key Implementation Details

### Server Components

1. **Data Structures**
   - `tcpClients`: Map of TCP connections to usernames
   - `messageHistory`: Circular buffer (max 100 messages)
   - `userStatuses`: Map of usernames to status info
   - `udpClients`: Map of usernames to UDP addresses

2. **Concurrency**
   - `sync.RWMutex` for thread-safe operations
   - Separate goroutines for each TCP client
   - Dedicated goroutines for TCP and UDP servers

3. **Message Broadcasting**
   - TCP: Broadcast to all connected clients except sender
   - UDP: Broadcast status to all registered UDP clients

### Client Components

1. **Connection Management**
   - Dual connections (TCP + UDP)
   - Automatic reconnection handling
   - Graceful shutdown on disconnect

2. **Concurrent Operations**
   - TCP message receiver (goroutine)
   - UDP status receiver (goroutine)
   - Heartbeat sender (goroutine)
   - Main input handler

3. **User Experience**
   - Real-time message display
   - Status notifications
   - Command-line interface
   - Message history on join

## Testing Scenarios

### Scenario 1: Basic Chat
1. Start server
2. Connect 2 clients (Alice, Bob)
3. Send messages back and forth
4. Verify both clients receive messages

### Scenario 2: Status Updates
1. Alice types `/status typing`
2. Bob should see: "Status Update: Alice is typing"
3. Alice types `/status away`
4. Bob should see: "Status Update: Alice is away"

### Scenario 3: Message History
1. Send several messages
2. Connect a new client (Charlie)
3. Charlie receives last 10 messages automatically
4. Charlie can request more: `/history 20`

### Scenario 4: Disconnection
1. Connect multiple clients
2. Disconnect one client
3. Server logs disconnection
4. Other clients continue chatting normally

## Real-World Applications

This architecture is used by:
- **WhatsApp**: TCP for messages, UDP for typing indicators
- **Discord**: TCP for chat, UDP for voice
- **Microsoft Teams**: TCP for messages, UDP for presence
- **LinkedIn**: TCP for posts, UDP for "online now" status

## Troubleshooting

**Port already in use:**
```powershell
# Find process using the port
netstat -ano | findstr :9000
# Kill the process (replace PID)
taskkill /PID <PID> /F
```

**Cannot connect:**
- Ensure server is running first
- Check firewall settings
- Verify ports 9000 and 9001 are available

**Messages not broadcasting:**
- Check mutex locks are properly released
- Verify goroutines are running
- Check for panic/errors in server logs
