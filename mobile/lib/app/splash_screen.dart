import 'package:flutter/material.dart';

import 'theme.dart';
import 'widgets/brand_logo.dart';

/// Shown while the app determines whether a stored session is still valid.
class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final p = Theme.of(context).palette;
    return Scaffold(
      backgroundColor: p.loftBg,
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(22),
                boxShadow: [
                  BoxShadow(
                    color: p.brandGreen.withValues(alpha: 0.25),
                    blurRadius: 32,
                    offset: const Offset(0, 12),
                  ),
                ],
              ),
              child: const BrandLogo(size: 72),
            ),
            const SizedBox(height: 32),
            SizedBox(
              height: 22,
              width: 22,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: p.brandGreen,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
