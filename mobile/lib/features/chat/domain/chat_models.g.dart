// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'chat_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$FertilizerProductImpl _$$FertilizerProductImplFromJson(
  Map<String, dynamic> json,
) => _$FertilizerProductImpl(
  id: json['id'] as String? ?? '',
  slug: json['slug'] as String? ?? '',
  name: json['name'] as String? ?? '',
  shortDesc: json['short_desc'] as String? ?? '',
  imageUrl: json['image_url'] as String? ?? '',
  deeplinkUrl: json['deeplink_url'] as String? ?? '',
);

Map<String, dynamic> _$$FertilizerProductImplToJson(
  _$FertilizerProductImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'slug': instance.slug,
  'name': instance.name,
  'short_desc': instance.shortDesc,
  'image_url': instance.imageUrl,
  'deeplink_url': instance.deeplinkUrl,
};

_$ContentBlockImpl _$$ContentBlockImplFromJson(Map<String, dynamic> json) =>
    _$ContentBlockImpl(
      type: json['type'] as String,
      text: json['text'] as String? ?? '',
      storageKey: json['storage_key'] as String? ?? '',
      durationMs: (json['duration_ms'] as num?)?.toInt() ?? 0,
      products:
          (json['products'] as List<dynamic>?)
              ?.map(
                (e) => FertilizerProduct.fromJson(e as Map<String, dynamic>),
              )
              .toList() ??
          const <FertilizerProduct>[],
    );

Map<String, dynamic> _$$ContentBlockImplToJson(_$ContentBlockImpl instance) =>
    <String, dynamic>{
      'type': instance.type,
      'text': instance.text,
      'storage_key': instance.storageKey,
      'duration_ms': instance.durationMs,
      'products': instance.products,
    };

_$ChatMessageImpl _$$ChatMessageImplFromJson(Map<String, dynamic> json) =>
    _$ChatMessageImpl(
      id: json['id'] as String,
      role: $enumDecode(_$MessageRoleEnumMap, json['role']),
      status: $enumDecode(_$MessageStatusEnumMap, json['status']),
      createdAt: DateTime.parse(json['created_at'] as String),
      content:
          (json['content'] as List<dynamic>?)
              ?.map((e) => ContentBlock.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const <ContentBlock>[],
    );

Map<String, dynamic> _$$ChatMessageImplToJson(_$ChatMessageImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'role': _$MessageRoleEnumMap[instance.role]!,
      'status': _$MessageStatusEnumMap[instance.status]!,
      'created_at': instance.createdAt.toIso8601String(),
      'content': instance.content,
    };

const _$MessageRoleEnumMap = {
  MessageRole.user: 'user',
  MessageRole.assistant: 'assistant',
  MessageRole.system: 'system',
};

const _$MessageStatusEnumMap = {
  MessageStatus.pending: 'pending',
  MessageStatus.complete: 'complete',
  MessageStatus.cancelled: 'cancelled',
  MessageStatus.failed: 'failed',
};

_$ConversationPageImpl _$$ConversationPageImplFromJson(
  Map<String, dynamic> json,
) => _$ConversationPageImpl(
  messages:
      (json['messages'] as List<dynamic>?)
          ?.map((e) => ChatMessage.fromJson(e as Map<String, dynamic>))
          .toList() ??
      const <ChatMessage>[],
  nextCursor: json['next_cursor'] as String?,
);

Map<String, dynamic> _$$ConversationPageImplToJson(
  _$ConversationPageImpl instance,
) => <String, dynamic>{
  'messages': instance.messages,
  'next_cursor': instance.nextCursor,
};
