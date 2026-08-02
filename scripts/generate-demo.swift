#!/usr/bin/env swift

import AppKit
import ImageIO
import UniformTypeIdentifiers

private let canvasWidth = 1200
private let canvasHeight = 880
private let lineHeight: CGFloat = 30

private enum Tone {
    case normal
    case muted
    case command
    case success
    case warning
    case danger
    case accent
}

private struct TerminalLine {
    let text: String
    let tone: Tone
    let bold: Bool

    init(_ text: String, _ tone: Tone = .normal, bold: Bool = false) {
        self.text = text
        self.tone = tone
        self.bold = bold
    }
}

private func color(_ hex: UInt32) -> NSColor {
    NSColor(
        calibratedRed: CGFloat((hex >> 16) & 0xff) / 255,
        green: CGFloat((hex >> 8) & 0xff) / 255,
        blue: CGFloat(hex & 0xff) / 255,
        alpha: 1
    )
}

private func foreground(_ tone: Tone) -> NSColor {
    switch tone {
    case .normal:
        return color(0xe6edf3)
    case .muted:
        return color(0x8b949e)
    case .command:
        return color(0x79c0ff)
    case .success:
        return color(0x56d364)
    case .warning:
        return color(0xe3b341)
    case .danger:
        return color(0xff7b72)
    case .accent:
        return color(0xd2a8ff)
    }
}

private func drawText(
    _ text: String,
    at point: NSPoint,
    color: NSColor,
    size: CGFloat,
    bold: Bool = false
) {
    let weight: NSFont.Weight = bold ? .semibold : .regular
    let font = NSFont.monospacedSystemFont(ofSize: size, weight: weight)
    let attributes: [NSAttributedString.Key: Any] = [
        .font: font,
        .foregroundColor: color,
    ]
    (text as NSString).draw(at: point, withAttributes: attributes)
}

private func render(title: String, lines: [TerminalLine]) -> CGImage {
    guard let bitmap = NSBitmapImageRep(
        bitmapDataPlanes: nil,
        pixelsWide: canvasWidth,
        pixelsHigh: canvasHeight,
        bitsPerSample: 8,
        samplesPerPixel: 4,
        hasAlpha: true,
        isPlanar: false,
        colorSpaceName: .deviceRGB,
        bytesPerRow: 0,
        bitsPerPixel: 0
    ), let graphics = NSGraphicsContext(bitmapImageRep: bitmap) else {
        fatalError("could not create bitmap context")
    }

    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = graphics

    color(0x070b10).setFill()
    NSRect(x: 0, y: 0, width: canvasWidth, height: canvasHeight).fill()

    let terminal = NSRect(x: 20, y: 20, width: 1160, height: canvasHeight - 40)
    color(0x0d1117).setFill()
    NSBezierPath(roundedRect: terminal, xRadius: 16, yRadius: 16).fill()

    let titleBar = NSRect(x: 20, y: canvasHeight - 80, width: 1160, height: 60)
    color(0x161b22).setFill()
    NSBezierPath(
        roundedRect: NSRect(x: titleBar.minX, y: titleBar.minY, width: titleBar.width, height: 60),
        xRadius: 16,
        yRadius: 16
    ).fill()
    color(0x161b22).setFill()
    NSRect(x: 20, y: canvasHeight - 80, width: 1160, height: 28).fill()

    for (x, value) in [(50, 0xff5f56), (78, 0xffbd2e), (106, 0x27c93f)] {
        color(UInt32(value)).setFill()
        NSBezierPath(ovalIn: NSRect(x: x, y: canvasHeight - 54, width: 14, height: 14)).fill()
    }

    let titleSize = (title as NSString).size(withAttributes: [
        .font: NSFont.monospacedSystemFont(ofSize: 17, weight: .medium),
    ])
    let titleX = CGFloat(canvasWidth) / 2 - titleSize.width / 2
    let titleY = CGFloat(canvasHeight - 58)
    drawText(
        title,
        at: NSPoint(x: titleX, y: titleY),
        color: color(0x8b949e),
        size: 17,
        bold: true
    )

    var y: CGFloat = CGFloat(canvasHeight - 120)
    for line in lines {
        drawText(
            line.text,
            at: NSPoint(x: 52, y: y),
            color: foreground(line.tone),
            size: 20,
            bold: line.bold
        )
        y -= lineHeight
    }

    graphics.flushGraphics()
    NSGraphicsContext.restoreGraphicsState()

    guard let image = bitmap.cgImage else {
        fatalError("could not create image")
    }
    return image
}

private func writePNG(_ image: CGImage, to url: URL) {
    guard let destination = CGImageDestinationCreateWithURL(
        url as CFURL,
        UTType.png.identifier as CFString,
        1,
        nil
    ) else {
        fatalError("could not create PNG destination")
    }
    CGImageDestinationAddImage(destination, image, nil)
    guard CGImageDestinationFinalize(destination) else {
        fatalError("could not write \(url.path)")
    }
}

private func writeGIF(_ images: [CGImage], delays: [Double], to url: URL) {
    guard images.count == delays.count else {
        fatalError("GIF images and delays differ")
    }
    guard let destination = CGImageDestinationCreateWithURL(
        url as CFURL,
        UTType.gif.identifier as CFString,
        images.count,
        nil
    ) else {
        fatalError("could not create GIF destination")
    }

    let globalProperties: CFDictionary = [
        kCGImagePropertyGIFDictionary: [
            kCGImagePropertyGIFLoopCount: 0,
        ],
    ] as CFDictionary
    CGImageDestinationSetProperties(destination, globalProperties)

    for (image, delay) in zip(images, delays) {
        let frameProperties: CFDictionary = [
            kCGImagePropertyGIFDictionary: [
                kCGImagePropertyGIFDelayTime: delay,
                kCGImagePropertyGIFUnclampedDelayTime: delay,
            ],
        ] as CFDictionary
        CGImageDestinationAddImage(destination, image, frameProperties)
    }

    guard CGImageDestinationFinalize(destination) else {
        fatalError("could not write \(url.path)")
    }
}

private let statusLines: [TerminalLine] = [
    TerminalLine("$ cd ~/work/acme/backend", .command, bold: true),
    TerminalLine("$ agentwho status", .command, bold: true),
    TerminalLine(""),
    TerminalLine("AgentWho status", .accent, bold: true),
    TerminalLine(""),
    TerminalLine("Directory:         /Users/example/work/acme/backend"),
    TerminalLine("Git root:          /Users/example/work/acme/backend"),
    TerminalLine("Repository:        github.com/acme/backend"),
    TerminalLine("Matched by:        organization github.com/acme"),
    TerminalLine(""),
    TerminalLine("Expected profile:  work", .success, bold: true),
    TerminalLine("Current profile:   work", .success, bold: true),
    TerminalLine("Safety mode:       confirm"),
    TerminalLine(""),
    TerminalLine("Claude command:    AgentWho active"),
    TerminalLine("Codex command:     AgentWho active"),
    TerminalLine(""),
    TerminalLine("✓ Ready", .success, bold: true),
]

private let automaticLines = statusLines + [
    TerminalLine(""),
    TerminalLine("$ claude", .command, bold: true),
]

private let mismatchBase: [TerminalLine] = [
    TerminalLine("$ agentwho use personal", .command, bold: true),
    TerminalLine("✓ Using profile \"personal\" in this shell.", .success),
    TerminalLine("⚠ This directory expects profile \"work\". Safety mode \"confirm\" will apply.", .warning),
    TerminalLine(""),
    TerminalLine("$ codex", .command, bold: true),
    TerminalLine(""),
    TerminalLine("Codex profile mismatch", .warning, bold: true),
    TerminalLine(""),
    TerminalLine("Repository:        github.com/acme/backend"),
    TerminalLine("Expected profile:  work", .success, bold: true),
    TerminalLine("Current profile:   personal", .danger, bold: true),
    TerminalLine(""),
    TerminalLine("Risk:", .warning, bold: true),
    TerminalLine("Company source code could be sent through your personal account.", .danger),
]

private let mismatchLines = mismatchBase + [
    TerminalLine(""),
    TerminalLine("What would you like to do?", .accent, bold: true),
    TerminalLine(""),
    TerminalLine("Use ↑/↓ to move and Enter to select.", .muted),
    TerminalLine("❯ Switch to profile \"work\" (recommended)", .success, bold: true),
    TerminalLine("    Use the profile expected for this directory.", .muted),
    TerminalLine("  Continue with profile \"personal\""),
    TerminalLine("    Ignore this binding for this command.", .muted),
    TerminalLine("  Cancel"),
    TerminalLine("    Do not start Codex.", .muted),
]

private let switchedLines = mismatchBase + [
    TerminalLine(""),
    TerminalLine("✓ Selected: Switch to profile \"work\" (recommended)", .success, bold: true),
    TerminalLine(""),
    TerminalLine("Using profile \"work\" for this command.", .success, bold: true),
    TerminalLine(""),
    TerminalLine("Codex starts with the expected work account.", .muted),
]

private let doctorLines: [TerminalLine] = [
    TerminalLine("$ agentwho doctor", .command, bold: true),
    TerminalLine(""),
    TerminalLine("AgentWho doctor", .accent, bold: true),
    TerminalLine(""),
    TerminalLine("Configuration", .normal, bold: true),
    TerminalLine("  ✓ Configuration is valid", .success),
    TerminalLine("  ✓ Data directory permissions are secure", .success),
    TerminalLine("  ✓ Profile personal / Claude directory is available", .success),
    TerminalLine("  ✓ Profile personal / Codex directory is available", .success),
    TerminalLine(""),
    TerminalLine("Automatic profile selection", .normal, bold: true),
    TerminalLine("  ✓ AgentWho command for Claude Code is installed", .success),
    TerminalLine("  ✓ AgentWho command for Codex is installed", .success),
    TerminalLine("  ✓ AgentWho comes before the official CLIs in PATH", .success),
    TerminalLine(""),
    TerminalLine("Official CLIs", .normal, bold: true),
    TerminalLine("  ✓ Claude Code: /opt/homebrew/bin/claude", .success),
    TerminalLine("  ✓ Codex: /opt/homebrew/bin/codex", .success),
    TerminalLine(""),
    TerminalLine("Result: Everything looks good.", .success, bold: true),
]

private let scriptURL = URL(fileURLWithPath: #filePath)
private let repositoryURL = scriptURL.deletingLastPathComponent().deletingLastPathComponent()
private let assetsURL = repositoryURL.appendingPathComponent("docs/assets", isDirectory: true)

try FileManager.default.createDirectory(at: assetsURL, withIntermediateDirectories: true)

let automatic = render(title: "AgentWho — automatic account selection", lines: automaticLines)
let status = render(title: "AgentWho — status", lines: statusLines)
let mismatch = render(title: "AgentWho — profile mismatch", lines: mismatchLines)
let switched = render(title: "AgentWho — safe switch", lines: switchedLines)
let doctor = render(title: "AgentWho — doctor", lines: doctorLines)

writePNG(status, to: assetsURL.appendingPathComponent("status.png"))
writePNG(mismatch, to: assetsURL.appendingPathComponent("mismatch.png"))
writePNG(switched, to: assetsURL.appendingPathComponent("switched.png"))
writePNG(doctor, to: assetsURL.appendingPathComponent("doctor.png"))
writeGIF(
    [automatic, mismatch, switched],
    delays: [3.2, 4.0, 3.2],
    to: assetsURL.appendingPathComponent("agentwho-demo.gif")
)

print("Generated terminal assets in \(assetsURL.path)")
