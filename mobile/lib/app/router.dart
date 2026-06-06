import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/application/auth_controller.dart';
import '../features/auth/presentation/email_request_screen.dart';
import '../features/auth/presentation/email_verify_screen.dart';
import '../features/auth/presentation/login_screen.dart';
import '../features/chat/presentation/chat_screen.dart';
import '../features/onboarding/application/onboarding_controller.dart';
import '../features/onboarding/presentation/onboarding_screen.dart';
import 'splash_screen.dart';

/// The app router. Redirects are driven by [authStatusProvider]: the splash is
/// shown until bootstrap resolves, unauthenticated users are confined to the
/// /login subtree, and authenticated users are sent to /chat.
final routerProvider = Provider<GoRouter>((ref) {
  final authRefresh = ValueNotifier<AuthStatus>(AuthStatus.unknown);
  ref.onDispose(authRefresh.dispose);
  ref.listen<AuthStatus>(
    authStatusProvider,
    (_, next) => authRefresh.value = next,
    fireImmediately: true,
  );

  final onboardingRefresh = ValueNotifier<OnboardingStatus>(
    OnboardingStatus.unknown,
  );
  ref.onDispose(onboardingRefresh.dispose);
  ref.listen<OnboardingStatus>(
    onboardingStatusProvider,
    (_, next) => onboardingRefresh.value = next,
    fireImmediately: true,
  );

  return GoRouter(
    initialLocation: '/splash',
    refreshListenable: Listenable.merge([authRefresh, onboardingRefresh]),
    redirect: (context, state) {
      final status = authRefresh.value;
      final onboarding = onboardingRefresh.value;
      final loc = state.matchedLocation;
      final atSplash = loc == '/splash';
      final atLogin = loc.startsWith('/login');
      final atOnboarding = loc == '/onboarding';

      switch (status) {
        case AuthStatus.unknown:
          return atSplash ? null : '/splash';
        case AuthStatus.unauthenticated:
          // Gate the one-time intro before the login subtree.
          switch (onboarding) {
            case OnboardingStatus.unknown:
              return atSplash ? null : '/splash';
            case OnboardingStatus.notSeen:
              return atOnboarding ? null : '/onboarding';
            case OnboardingStatus.seen:
              return atLogin ? null : '/login';
          }
        case AuthStatus.authenticated:
          return (atLogin || atSplash || atOnboarding) ? '/chat' : null;
      }
    },
    routes: [
      GoRoute(path: '/splash', builder: (_, _) => const SplashScreen()),
      GoRoute(path: '/onboarding', builder: (_, _) => const OnboardingScreen()),
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
