import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../l10n/generated/app_localizations.dart';
import '../../domain/chat_models.dart';

/// Renders a vertical stack of fertilizer recommendation cards inside an
/// assistant bubble (stage 5.3). Each card shows the product photo, name, short
/// description and a "Learn more" button that opens the deeplink in the browser.
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
    final l10n = AppLocalizations.of(context)!;

    return Card(
      margin: EdgeInsets.zero,
      clipBehavior: Clip.antiAlias,
      color: theme.colorScheme.surface,
      child: InkWell(
        onTap: _open,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (product.imageUrl.isNotEmpty) _image(theme),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    product.name,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  if (product.shortDesc.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      product.shortDesc,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                  const SizedBox(height: 8),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: FilledButton.tonalIcon(
                      onPressed: _open,
                      icon: const Icon(Icons.open_in_new, size: 18),
                      label: Text(l10n.fertilizerCardMore),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _image(ThemeData theme) {
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
            color: theme.colorScheme.surfaceContainerHighest,
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
          color: theme.colorScheme.surfaceContainerHighest,
          alignment: Alignment.center,
          child: Icon(
            Icons.local_florist_outlined,
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ),
    );
  }
}
