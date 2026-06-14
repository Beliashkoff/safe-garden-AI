import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/theme.dart';
import '../../../../l10n/generated/app_localizations.dart';
import '../../data/media_cache.dart';

/// Renders the photos of a user message. A single photo shows large (loft
/// style); multiple photos wrap into rounded tiles. Each [storageKeys] entry is
/// resolved to a local file via [mediaFileProvider] (cache hit for a just-sent
/// photo, lazy download for history). When [analysed] is true the first photo
/// carries a "Проанализировано" badge.
class MessagePhotos extends ConsumerWidget {
  const MessagePhotos({
    required this.storageKeys,
    this.analysed = false,
    super.key,
  });

  final List<String> storageKeys;
  final bool analysed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (storageKeys.length == 1) {
      return _single(context, ref, storageKeys.first);
    }
    return Wrap(
      spacing: 4,
      runSpacing: 4,
      alignment: WrapAlignment.end,
      children: [
        for (var i = 0; i < storageKeys.length; i++)
          _tile(context, ref, storageKeys[i], size: 120, badge: analysed && i == 0),
      ],
    );
  }

  Widget _single(BuildContext context, WidgetRef ref, String key) {
    final width = (MediaQuery.of(context).size.width * 0.62).clamp(180.0, 260.0);
    return _frame(
      context,
      ref,
      key,
      width: width,
      height: width * 0.75,
      badge: analysed,
    );
  }

  Widget _tile(
    BuildContext context,
    WidgetRef ref,
    String key, {
    required double size,
    required bool badge,
  }) {
    return _frame(context, ref, key, width: size, height: size, badge: badge);
  }

  Widget _frame(
    BuildContext context,
    WidgetRef ref,
    String key, {
    required double width,
    required double height,
    required bool badge,
  }) {
    final async = ref.watch(mediaFileProvider(key));
    return ClipRRect(
      borderRadius: BorderRadius.circular(AppRadius.photo),
      child: SizedBox(
        width: width,
        height: height,
        child: Stack(
          fit: StackFit.expand,
          children: [
            async.when(
              data: (file) => Image.file(
                file,
                fit: BoxFit.cover,
                errorBuilder: (context, _, _) =>
                    _placeholder(context, broken: true),
              ),
              loading: () => _placeholder(context, broken: false),
              error: (_, _) => _placeholder(context, broken: true),
            ),
            if (badge)
              Positioned(top: 10, left: 10, child: _AnalysedBadge()),
          ],
        ),
      ),
    );
  }

  Widget _placeholder(BuildContext context, {required bool broken}) {
    final theme = Theme.of(context);
    return Container(
      color: theme.palette.soft,
      alignment: Alignment.center,
      child: broken
          ? Icon(Icons.broken_image_outlined, color: theme.palette.textSubtle)
          : const SizedBox(
              height: 22,
              width: 22,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
    );
  }
}

class _AnalysedBadge extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.fromLTRB(7, 4, 9, 4),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.55),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: const BoxDecoration(
              color: Color(0xFF3FC56B),
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 5),
          Text(
            l10n.chatPhotoAnalyzed,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}
