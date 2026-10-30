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
            url: "https://github.com/Goodneo-Solutions/slocker-releases/releases/download/0.1.1/SLocker.xcframework.zip",
            checksum: "4489eb14e6beec597895dca3cba92aade97f0cbb2d65178d35f7d2dd8b44afd5"
        ),
    ]
)
