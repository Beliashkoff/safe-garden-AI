import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// The "ИИ Агроном" brand mark — a green rounded square with a white
/// sprout-in-hand glyph. Rendered from the bundled SVG so it stays crisp at
/// every size (header 24, AI author row 22, empty-state hero 64).
class BrandLogo extends StatelessWidget {
  const BrandLogo({super.key, this.size = 24});

  final double size;

  @override
  Widget build(BuildContext context) {
    return SvgPicture.asset(
      'assets/brand/agro-logo.svg',
      width: size,
      height: size,
      fit: BoxFit.contain,
    );
  }
}
