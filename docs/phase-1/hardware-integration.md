# Hardware Integration Strategy — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer

---

## 1. Overview

POS hardware integration is one of the hardest parts of building a web/mobile POS system because the web platform was not designed with hardware access in mind. This document covers:

1. **Thermal Printer (ESC/POS)** — receipt printing
2. **Cash Drawer** — open/close trigger
3. **Barcode Scanner** — product lookup
4. **Secondary Display** — customer-facing screen

Each hardware type has different connectivity methods across Web and Mobile, requiring an Anti-Corruption Layer (ACL) pattern in DDD terms — the application domain doesn't know or care if a printer is connected via USB, Bluetooth, or network.

---

## 2. Architecture: Anti-Corruption Layer

```
Application Domain (POS Core)
        │
        │  IReceiptPrinter interface
        │  ICashDrawer interface
        │  IBarcodeScanner interface
        ▼
Hardware ACL (Adapter Layer)
        │
        ├── Web Adapter
        │   ├── WebUSBPrinterAdapter
        │   ├── WebSerialPrinterAdapter
        │   ├── NetworkPrinterAdapter (TCP/IP)
        │   └── WebHIDBarcodeAdapter
        │
        └── Mobile Adapter (Flutter)
            ├── BluetoothPrinterAdapter
            ├── USBPrinterAdapter (Android)
            ├── NetworkPrinterAdapter
            └── CameraBarcodeScannerAdapter
```

Domain code calls `IReceiptPrinter.Print(receipt)`. It never references USB, Bluetooth, or serial ports. The platform-specific adapter resolves the physical connection.

---

## 3. Thermal Printer (ESC/POS Protocol)

### 3.1 What is ESC/POS?

ESC/POS is a proprietary command language developed by Epson but adopted as an industry standard by virtually all thermal receipt printers. Commands are sequences of bytes — `ESC` (0x1B) followed by a control byte and optional data.

Common commands:
```
ESC @          (0x1B 0x40)     — Initialize printer
ESC ! n        (0x1B 0x21 n)   — Select print mode (bold, double-width, etc.)
GS V m         (0x1D 0x56 m)   — Cut paper
ESC d n        (0x1B 0x64 n)   — Feed n lines
ESC a n        (0x1B 0x61 n)   — Justification (0=left, 1=center, 2=right)
GS k m n       (0x1D 0x6B m n) — Print barcode
```

### 3.2 Receipt Data Model (Go)

Build a printer-agnostic receipt model in the domain layer:

```go
// domain/receipt/receipt.go
type Receipt struct {
    Header    ReceiptHeader
    Lines     []ReceiptLine
    Totals    ReceiptTotals
    Payment   ReceiptPayment
    Footer    ReceiptFooter
    QRContent string    // QR code data (e.g., digital receipt URL)
}

type ReceiptLine struct {
    Name     string
    Quantity int
    Price    decimal.Decimal
    Subtotal decimal.Decimal
    Addons   []string
}
```

### 3.3 Go ESC/POS Encoder

```go
// infrastructure/printer/escpos/encoder.go
type Encoder struct {
    buf bytes.Buffer
}

func (e *Encoder) Initialize() {
    e.buf.Write([]byte{0x1B, 0x40}) // ESC @
}

func (e *Encoder) SetBold(on bool) {
    if on {
        e.buf.Write([]byte{0x1B, 0x21, 0x08}) // bold on
    } else {
        e.buf.Write([]byte{0x1B, 0x21, 0x00}) // bold off
    }
}

func (e *Encoder) SetAlign(align Alignment) {
    e.buf.Write([]byte{0x1B, 0x61, byte(align)})
}

func (e *Encoder) CutPaper() {
    e.buf.Write([]byte{0x1D, 0x56, 0x42, 0x00}) // GS V B — full cut
}

func (e *Encoder) PrintQR(data string, size QRSize) {
    // QR code is a multi-step ESC/POS sequence
    // 1. Set model, 2. Set size, 3. Store data, 4. Print
    // ... (implementation details vary by printer model)
}

func (e *Encoder) Bytes() []byte {
    return e.buf.Bytes()
}
```

### 3.4 Web: Connection Methods

**Priority order for Web POS:**

| Method | Pros | Cons | Browser Support |
|---|---|---|---|
| **WebUSB** | Reliable, fast, no driver needed | Chrome/Edge only, requires HTTPS, user gesture | Chrome 61+ |
| **WebSerial** | Works with RS-232/COM port printers | Serial-to-USB adapter needed | Chrome 89+ |
| **Network (TCP via WebSocket)** | No browser restrictions, works on all browsers | Requires printer bridge service running locally | All browsers |
| **Web Bluetooth** | Wireless | Slow, unreliable for large receipts | Limited support |

**Recommended: Network Bridge (most compatible)**

A lightweight local service runs on the POS machine and exposes a WebSocket endpoint. The web app connects to `ws://localhost:9000/printer`. This is the most reliable approach and works on all browsers including Firefox and Safari.

```
Web App → WebSocket → Local Bridge Service (Go binary) → USB/Network Printer
```

```go
// Local bridge service (distributed with the POS setup package)
// infra/printer-bridge/main.go
func main() {
    hub := NewPrinterHub()
    http.HandleFunc("/printer", hub.HandleWebSocket)
    http.HandleFunc("/printer/discover", hub.HandleDiscover)
    log.Fatal(http.ListenAndServe("localhost:9000", nil))
}
```

**WebUSB for Chrome-only environments (Chromebook POS):**

```typescript
// apps/web/src/lib/hardware/printer/WebUSBPrinterAdapter.ts
export class WebUSBPrinterAdapter implements IReceiptPrinter {
  private device: USBDevice | null = null;

  async connect(): Promise<void> {
    this.device = await navigator.usb.requestDevice({
      filters: [
        { vendorId: 0x04b8 }, // Epson
        { vendorId: 0x0519 }, // Star Micronics
        { vendorId: 0x0fe6 }, // ICS Advent (generic thermal)
      ],
    });
    await this.device.open();
    await this.device.selectConfiguration(1);
    await this.device.claimInterface(0);
  }

  async print(data: Uint8Array): Promise<void> {
    if (!this.device) throw new Error('Printer not connected');
    await this.device.transferOut(1, data);
  }
}
```

### 3.5 Mobile: Flutter Integration

```dart
// Flutter: Bluetooth ESC/POS (thermal_printer package)
class BluetoothPrinterAdapter implements IReceiptPrinter {
  final PrinterBluetooth _device;

  @override
  Future<void> print(List<int> bytes) async {
    final generator = Generator(PaperSize.mm80, await CapabilityProfile.load());
    // bytes is the ESC/POS encoded receipt
    await PrinterBluetooth.instance.printTicket(bytes);
  }
}
```

**Recommended Flutter packages:**
- `esc_pos_utils` — ESC/POS command generation
- `flutter_pos_printer_platform` — Cross-platform (USB, Network, Bluetooth) printer abstraction

---

## 4. Cash Drawer

### 4.1 How Cash Drawers Work

Cash drawers are almost always connected to the thermal printer (not directly to the computer). The printer has a RJ11 port, and the cash drawer plugs into it. To open the drawer, you send a specific ESC/POS command to the printer:

```
ESC p m t1 t2
0x1B 0x70 0x00 0x19 0xFA   — Open drawer on port 0 (pin 2)
0x1B 0x70 0x01 0x19 0xFA   — Open drawer on port 1 (pin 5)
```

```go
// infrastructure/printer/escpos/encoder.go
func (e *Encoder) OpenCashDrawer(port CashDrawerPort) {
    e.buf.Write([]byte{0x1B, 0x70, byte(port), 0x19, 0xFA})
}
```

### 4.2 Integration Pattern

Cash drawer opening is a **side effect of payment completion**, not a separate hardware command:

```go
// application/command/complete_payment.go
func (h *CompletePaymentHandler) Handle(ctx context.Context, cmd CompletePaymentCommand) error {
    // ... payment processing logic ...

    // Publish domain event — printer/drawer adapter listens
    h.eventBus.Publish(ctx, PaymentCompletedEvent{
        OrderID:        cmd.OrderID,
        PaymentMethod:  cmd.Method,
        ShouldOpenDrawer: cmd.Method == PaymentMethodCash,
        Receipt:        receipt,
    })
    return nil
}

// infrastructure/hardware/event_handler.go
func (h *HardwareEventHandler) OnPaymentCompleted(evt PaymentCompletedEvent) {
    printer := h.printerRegistry.GetForBranch(evt.BranchID)

    receiptBytes := h.encoder.Encode(evt.Receipt)
    if evt.ShouldOpenDrawer {
        h.encoder.OpenCashDrawer(CashDrawerPort0) // added to receipt bytes
    }
    h.encoder.CutPaper()

    printer.Print(h.encoder.Bytes())
}
```

---

## 5. Barcode Scanner

### 5.1 Input Modes

| Mode | Web | Mobile | Notes |
|---|---|---|---|
| **USB HID (keyboard emulation)** | ✅ Native (reads as keyboard input) | ✅ Android OTG | Most common scanner mode |
| **WebHID API** | ✅ Chrome only | ❌ | Low-level HID access |
| **Camera** | ✅ (getUserMedia) | ✅ Flutter camera | Fallback, slower |
| **Bluetooth HID** | ✅ Web Bluetooth | ✅ Flutter BT | Wireless scanners |

### 5.2 Web: USB HID (Keyboard Emulation)

Most barcode scanners in USB mode act as a keyboard. They "type" the barcode value and append a configurable suffix (usually Enter or Tab). The simplest integration:

```typescript
// apps/web/src/lib/hardware/barcode/KeyboardBarcodeListener.ts
export class KeyboardBarcodeListener {
  private buffer = '';
  private lastKeyTime = 0;
  private readonly SCAN_TIMEOUT_MS = 50; // Scanners type very fast

  start(onScan: (barcode: string) => void): () => void {
    const handler = (e: KeyboardEvent) => {
      const now = Date.now();

      if (now - this.lastKeyTime > this.SCAN_TIMEOUT_MS) {
        this.buffer = ''; // Reset buffer if too much time has passed
      }

      this.lastKeyTime = now;

      if (e.key === 'Enter' && this.buffer.length > 3) {
        // Minimum barcode length to avoid false triggers
        onScan(this.buffer);
        this.buffer = '';
        e.preventDefault();
      } else if (e.key.length === 1) {
        this.buffer += e.key;
      }
    };

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }
}
```

**Important:** This listener should only be active when the POS search field is not focused to avoid capturing typed text as barcodes.

### 5.3 Web: Camera Barcode Scanner

Use **ZXing** or **QuaggaJS** for camera-based scanning:

```typescript
import { BrowserMultiFormatReader } from '@zxing/browser';

const reader = new BrowserMultiFormatReader();
const videoElement = document.getElementById('video') as HTMLVideoElement;

reader.decodeFromVideoDevice(null, videoElement, (result, error) => {
  if (result) {
    onScan(result.getText());
  }
});
```

### 5.4 Mobile: Flutter

```dart
// Use mobile_scanner package (fast, uses ML Kit / ZXing)
MobileScanner(
  onDetect: (capture) {
    final barcodes = capture.barcodes;
    for (final barcode in barcodes) {
      onScan(barcode.rawValue ?? '');
    }
  },
)
```

### 5.5 Barcode Lookup Flow

```
Scanner Input (barcode string)
        │
        ▼
Local Cache Check (Drift/SQLite on mobile, localStorage on web)
        │
        ├── Cache HIT → Return product immediately (< 10ms)
        │
        └── Cache MISS
                │
                ▼
        gRPC ProductService.LookupBySKU(barcode)
                │
                ▼
        Add to cart + Update local cache
```

Cache product catalog locally on mobile for offline support. Sync catalog on login and periodically via background sync.

---

## 6. WebUSB / WebSerial / WebBluetooth Security

These are powerful browser APIs that require explicit user permission and HTTPS.

| API | Permission Model | HTTPS Required | Notes |
|---|---|---|---|
| WebUSB | User gesture + permission prompt | ✅ | Per-device permission, persisted |
| WebSerial | User gesture + permission prompt | ✅ | Per-port permission |
| WebBluetooth | User gesture + permission prompt | ✅ | Per-device, varies by OS |
| WebHID | User gesture + permission prompt | ✅ | Low-level HID access |

**Implementation Rule:** Always wrap hardware access in a `HardwarePermissionManager` that:
1. Checks if permission is already granted
2. Requests permission only on explicit user action (button click), never on page load
3. Gracefully degrades if the browser doesn't support the API

```typescript
export class HardwarePermissionManager {
  async requestPrinter(): Promise<PrinterPermission> {
    if (!('usb' in navigator)) {
      return { type: 'network' }; // Fallback to network bridge
    }
    try {
      const device = await navigator.usb.requestDevice({ filters: PRINTER_FILTERS });
      return { type: 'webusb', device };
    } catch {
      return { type: 'unsupported' };
    }
  }
}
```

---

## 7. Printer Discovery & Configuration UI

The POS setup screen must allow the cashier to:
1. Scan for available printers (network broadcast, USB enumeration)
2. Select and test a printer
3. Configure paper width (58mm / 80mm)
4. Print a test receipt

```typescript
// Printer discovery protocol
interface PrinterDiscovery {
  scanNetwork(subnet: string): Promise<NetworkPrinter[]>;  // mDNS / port 9100 scan
  scanUSB(): Promise<USBDevice[]>;
  scanBluetooth(): Promise<BluetoothDevice[]>;
}
```

Store printer configuration in the local settings (IndexedDB on web, SharedPreferences on mobile) so it persists across sessions without requiring re-setup.

---

## 8. Hardware Integration Testing Strategy

Hardware is hard to test in CI. Strategy:

| Test Type | Approach |
|---|---|
| Unit tests | Mock `IReceiptPrinter`, `IBarcodeScanner` interfaces — no real hardware needed |
| Integration tests | Use a virtual serial port (socat) to simulate printer responses |
| ESC/POS output tests | Snapshot test the byte arrays — compare against known-good receipts |
| Manual QA | Physical hardware lab with representative printers (Epson TM-T82X, Star TSP143, Generic 58mm) |

```go
// Example: ESC/POS encoder unit test
func TestEncodeReceipt_IncludesDrawerOpenCommand(t *testing.T) {
    enc := escpos.NewEncoder()
    enc.OpenCashDrawer(escpos.CashDrawerPort0)
    bytes := enc.Bytes()

    // Assert drawer open command is present
    assert.Contains(t, bytes, []byte{0x1B, 0x70, 0x00, 0x19, 0xFA})
}
```
