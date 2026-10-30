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
            url: "https://github.com/Goodneo-Solutions/slocker-releases/releases/download/0.1.0/SLocker.xcframework.zip",
            checksum: "abec6fb447cb158824c60973cc734fb5e5c5676e1551ba888036b9d66c4b1327"
        ),
    ]
)
