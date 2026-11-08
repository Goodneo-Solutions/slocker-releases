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
            url: "https://github.com/Goodneo-Solutions/slocker-releases/releases/download/0.2.2/SLocker.xcframework.zip",
            checksum: "dbae25b7b094f1dcf9f274ddb866fad2f74687bd94e003a664dfdfcadea3bd7b"
        ),
    ]
)
