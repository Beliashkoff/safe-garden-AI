import 'package:go_router/go_router.dart';

import '../features/auth/presentation/login_screen.dart';
import '../features/chat/presentation/chat_screen.dart';

final GoRouter appRouter = GoRouter(
  initialLocation: '/login',
  routes: [
    GoRoute(
      path: '/login',
      name: 'login',
      builder: (context, state) => const LoginScreen(),
    ),
    GoRoute(
      path: '/chat',
      name: 'chat',
      builder: (context, state) => const ChatScreen(),
    ),
  ],
);
