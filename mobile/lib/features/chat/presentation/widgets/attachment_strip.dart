import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/generated/app_localizations.dart';
import '../../application/message_composer.dart';

/// Horizontal strip of staged photos shown above the input bar. Each tile shows
/// the compressed thumbnail with a remove button; during upload it overlays the
/// progress, and on failure a warning.
class AttachmentStrip extends ConsumerWidget {
  const AttachmentStrip({super.key});

  static const double _tile = 72;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(messageComposerProvider);
    if (!state.hasAttachments) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      height: _tile + 16,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        itemCount: state.attachments.length,
        separatorBuilder: (_, _) => const SizedBox(width: 8),
        itemBuilder: (context, i) =>
            _tileWidget(context, ref, l10n, state.attachments[i], state.uploading),
      ),
    );
  }

  Widget _tileWidget(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    PhotoAttachment att,
    bool uploading,
  ) {
    final theme = Theme.of(context);
    return Stack(
      clipBehavior: Clip.none,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(10),
          child: Image.file(
            att.file,
            width: _tile,
            height: _tile,
            fit: BoxFit.cover,
          ),
        ),
        if (att.status == AttachStatus.uploading)
          _overlay(
            child: SizedBox(
              height: 24,
              width: 24,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                value: att.progress > 0 ? att.progress : null,
              ),
            ),
          ),
        if (att.status == AttachStatus.failed)
          _overlay(
            child: Icon(
              Icons.error_outline,
              color: theme.colorScheme.onError,
            ),
          ),
        if (!uploading)
          Positioned(
            top: -8,
            right: -8,
            child: _RemoveButton(
              tooltip: l10n.chatRemovePhoto,
              onTap: () =>
                  ref.read(messageComposerProvider.notifier).remove(att.localId),
            ),
          ),
      ],
    );
  }

  Widget _overlay({required Widget child}) {
    return Positioned.fill(
      child: ClipRRect(
        borderRadius: BorderRadius.circular(10),
        child: ColoredBox(
          color: Colors.black45,
          child: Center(child: child),
        ),
      ),
    );
  }
}

class _RemoveButton extends StatelessWidget {
  const _RemoveButton({required this.tooltip, required this.onTap});

  final String tooltip;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Tooltip(
      message: tooltip,
      child: InkWell(
        onTap: onTap,
        customBorder: const CircleBorder(),
        child: CircleAvatar(
          radius: 11,
          backgroundColor: theme.colorScheme.surface,
          child: Icon(
            Icons.close,
            size: 15,
            color: theme.colorScheme.onSurface,
          ),
        ),
      ),
    );
  }
}
