import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/application/auth_controller.dart';
import '../features/auth/presentation/email_request_screen.dart';
import '../features/auth/presentation/email_verify_screen.dart';
import '../features/auth/presentation/login_screen.dart';
import '../features/chat/presentation/chat_screen.dart';
import 'splash_screen.dart';

/// The app router. Redirects are driven by [authStatusProvider]: the splash is
/// shown until bootstrap resolves, unauthenticated users are confined to the
/// /login subtree, and authenticated users are sent to /chat.
final routerProvider = Provider<GoRouter>((ref) {
  final refresh = ValueNotifier<AuthStatus>(AuthStatus.unknown);
  ref.onDispose(refresh.dispose);
  ref.listen<AuthStatus>(
    authStatusProvider,
    (_, next) => refresh.value = next,
    fireImmediately: true,
  );

  return GoRouter(
    initialLocation: '/splash',
    refreshListenable: refresh,
    redirect: (context, state) {
      final status = refresh.value;
      final loc = state.matchedLocation;
      final atSplash = loc == '/splash';
      final atLogin = loc.startsWith('/login');

      switch (status) {
        case AuthStatus.unknown:
          return atSplash ? null : '/splash';
        case AuthStatus.unauthenticated:
          return atLogin ? null : '/login';
        case AuthStatus.authenticated:
          return (atLogin || atSplash) ? '/chat' : null;
      }
    },
    routes: [
      GoRoute(path: '/splash', builder: (_, _) => const SplashScreen()),
      GoRoute(
        path: '/login',
        builder: (_, _) => const LoginScreen(),
        routes: [
          GoRoute(
            path: 'email',
            builder: (_, _) => const EmailRequestScreen(),
            routes: [
              GoRoute(
                path: 'verify',
                builder: (_, state) =>
                    EmailVerifyScreen(email: state.extra as String? ?? ''),
              ),
            ],
          ),
        ],
      ),
      GoRoute(path: '/chat', builder: (_, _) => const ChatScreen()),
    ],
  );
});
