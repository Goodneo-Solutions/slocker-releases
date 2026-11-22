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
            url: "https://github.com/Goodneo-Solutions/slocker-releases/releases/download/0.2.6/SLocker.xcframework.zip",
            checksum: "7cc0686ce2f95d6fa4c39daa903dcb222bfb91359e1576377b2c5f39e90dfecf"
        ),
    ]
)
