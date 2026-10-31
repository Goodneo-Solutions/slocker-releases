// swift-tools-version: 5.9
// sLocker for iOS and macOS, as a prebuilt XCFramework. Documentation: https://open.superroute.ca/developer
import PackageDescription

let package = Package(
    name: "SLocker",
    platforms: [.iOS(.v15), .macOS(.v12)],
    products: [
        .library(name: "SLocker", targets: ["SLocker"]),
    ],
    targets: [
        .binaryTarget(
            name: "SLocker",
            url: "https://github.com/Goodneo-Solutions/slocker-releases/releases/download/0.2.0/SLocker.xcframework.zip",
            checksum: "2dcf4da06c1fe4d8acaa22f32b5cee4ee11df379746f99dcf3e8af65e1314573"
        ),
    ]
)
