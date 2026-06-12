import Flutter
import UIKit
import vkid_flutter_sdk

class SceneDelegate: FlutterSceneDelegate {
  // VK ID deep-link return (vk{client_id}://) arrives here under the scene
  // lifecycle; hand it to the VK ID SDK first, then to Flutter plugins.
  override func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
    for context in URLContexts where VkidFlutterSdkPlugin.vkid.open(url: context.url) {
      return
    }
    super.scene(scene, openURLContexts: URLContexts)
  }
}
