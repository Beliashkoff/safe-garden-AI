import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../app/theme.dart';
import '../../../../l10n/generated/app_localizations.dart';
import '../../domain/chat_models.dart';

/// Renders a vertical stack of fertilizer recommendation cards under an
/// assistant answer (stage 5.3). Each card shows the product photo, name, short
/// description and a "Learn more" action that opens the deeplink in the browser.
/// Catalog images are public URLs (not presigned storage), so they load via
/// [Image.network] directly. [onOpen] reports a tap for internal analytics.
class FertilizerCardList extends StatelessWidget {
  const FertilizerCardList({required this.products, this.onOpen, super.key});

  final List<FertilizerProduct> products;
  final void Function(FertilizerProduct product)? onOpen;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        for (final p in products)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: _FertilizerCard(product: p, onOpen: onOpen),
          ),
      ],
    );
  }
}

class _FertilizerCard extends StatelessWidget {
  const _FertilizerCard({required this.product, this.onOpen});

  final FertilizerProduct product;
  final void Function(FertilizerProduct product)? onOpen;

  Future<void> _open() async {
    onOpen?.call(product);
    final url = product.deeplinkUrl;
    if (url.isEmpty) {
      return;
    }
    final uri = Uri.tryParse(url);
    if (uri == null) {
      return;
    }
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final l10n = AppLocalizations.of(context)!;

    return Material(
      color: p.softer,
      borderRadius: BorderRadius.circular(AppRadius.card),
      child: InkWell(
        onTap: _open,
        borderRadius: BorderRadius.circular(AppRadius.card),
        child: Container(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppRadius.card),
            border: Border.all(color: p.border),
          ),
          clipBehavior: Clip.antiAlias,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (product.imageUrl.isNotEmpty) _image(p),
              Padding(
                padding: const EdgeInsets.all(14),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(product.name, style: theme.textTheme.titleMedium),
                    if (product.shortDesc.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        product.shortDesc,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: p.textMuted,
                        ),
                      ),
                    ],
                    const SizedBox(height: 12),
                    _moreButton(p, theme, l10n),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _moreButton(AppPalette p, ThemeData theme, AppLocalizations l10n) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Material(
        color: p.brandGreenSoft,
        borderRadius: BorderRadius.circular(999),
        child: InkWell(
          onTap: _open,
          borderRadius: BorderRadius.circular(999),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  l10n.fertilizerCardMore,
                  style: theme.textTheme.labelLarge?.copyWith(
                    color: p.brandGreen,
                  ),
                ),
                const SizedBox(width: 6),
                Icon(Icons.open_in_new_rounded, size: 16, color: p.brandGreen),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _image(AppPalette p) {
    return AspectRatio(
      aspectRatio: 16 / 9,
      child: Image.network(
        product.imageUrl,
        fit: BoxFit.cover,
        loadingBuilder: (context, child, progress) {
          if (progress == null) {
            return child;
          }
          return Container(
            color: p.soft,
            child: const Center(
              child: SizedBox(
                height: 20,
                width: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          );
        },
        errorBuilder: (context, _, _) => Container(
          color: p.soft,
          alignment: Alignment.center,
          child: Icon(Icons.local_florist_outlined, color: p.textSubtle),
        ),
      ),
    );
  }
}
