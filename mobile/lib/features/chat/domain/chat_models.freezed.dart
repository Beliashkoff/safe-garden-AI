// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'chat_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

FertilizerProduct _$FertilizerProductFromJson(Map<String, dynamic> json) {
  return _FertilizerProduct.fromJson(json);
}

/// @nodoc
mixin _$FertilizerProduct {
  String get id => throw _privateConstructorUsedError;
  String get slug => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  @JsonKey(name: 'short_desc')
  String get shortDesc => throw _privateConstructorUsedError;
  @JsonKey(name: 'image_url')
  String get imageUrl => throw _privateConstructorUsedError;
  @JsonKey(name: 'deeplink_url')
  String get deeplinkUrl => throw _privateConstructorUsedError;

  /// Serializes this FertilizerProduct to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FertilizerProduct
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FertilizerProductCopyWith<FertilizerProduct> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FertilizerProductCopyWith<$Res> {
  factory $FertilizerProductCopyWith(
    FertilizerProduct value,
    $Res Function(FertilizerProduct) then,
  ) = _$FertilizerProductCopyWithImpl<$Res, FertilizerProduct>;
  @useResult
  $Res call({
    String id,
    String slug,
    String name,
    @JsonKey(name: 'short_desc') String shortDesc,
    @JsonKey(name: 'image_url') String imageUrl,
    @JsonKey(name: 'deeplink_url') String deeplinkUrl,
  });
}

/// @nodoc
class _$FertilizerProductCopyWithImpl<$Res, $Val extends FertilizerProduct>
    implements $FertilizerProductCopyWith<$Res> {
  _$FertilizerProductCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FertilizerProduct
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? slug = null,
    Object? name = null,
    Object? shortDesc = null,
    Object? imageUrl = null,
    Object? deeplinkUrl = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            slug: null == slug
                ? _value.slug
                : slug // ignore: cast_nullable_to_non_nullable
                      as String,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            shortDesc: null == shortDesc
                ? _value.shortDesc
                : shortDesc // ignore: cast_nullable_to_non_nullable
                      as String,
            imageUrl: null == imageUrl
                ? _value.imageUrl
                : imageUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            deeplinkUrl: null == deeplinkUrl
                ? _value.deeplinkUrl
                : deeplinkUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FertilizerProductImplCopyWith<$Res>
    implements $FertilizerProductCopyWith<$Res> {
  factory _$$FertilizerProductImplCopyWith(
    _$FertilizerProductImpl value,
    $Res Function(_$FertilizerProductImpl) then,
  ) = __$$FertilizerProductImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    String slug,
    String name,
    @JsonKey(name: 'short_desc') String shortDesc,
    @JsonKey(name: 'image_url') String imageUrl,
    @JsonKey(name: 'deeplink_url') String deeplinkUrl,
  });
}

/// @nodoc
class __$$FertilizerProductImplCopyWithImpl<$Res>
    extends _$FertilizerProductCopyWithImpl<$Res, _$FertilizerProductImpl>
    implements _$$FertilizerProductImplCopyWith<$Res> {
  __$$FertilizerProductImplCopyWithImpl(
    _$FertilizerProductImpl _value,
    $Res Function(_$FertilizerProductImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FertilizerProduct
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? slug = null,
    Object? name = null,
    Object? shortDesc = null,
    Object? imageUrl = null,
    Object? deeplinkUrl = null,
  }) {
    return _then(
      _$FertilizerProductImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        slug: null == slug
            ? _value.slug
            : slug // ignore: cast_nullable_to_non_nullable
                  as String,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        shortDesc: null == shortDesc
            ? _value.shortDesc
            : shortDesc // ignore: cast_nullable_to_non_nullable
                  as String,
        imageUrl: null == imageUrl
            ? _value.imageUrl
            : imageUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        deeplinkUrl: null == deeplinkUrl
            ? _value.deeplinkUrl
            : deeplinkUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FertilizerProductImpl implements _FertilizerProduct {
  const _$FertilizerProductImpl({
    this.id = '',
    this.slug = '',
    this.name = '',
    @JsonKey(name: 'short_desc') this.shortDesc = '',
    @JsonKey(name: 'image_url') this.imageUrl = '',
    @JsonKey(name: 'deeplink_url') this.deeplinkUrl = '',
  });

  factory _$FertilizerProductImpl.fromJson(Map<String, dynamic> json) =>
      _$$FertilizerProductImplFromJson(json);

  @override
  @JsonKey()
  final String id;
  @override
  @JsonKey()
  final String slug;
  @override
  @JsonKey()
  final String name;
  @override
  @JsonKey(name: 'short_desc')
  final String shortDesc;
  @override
  @JsonKey(name: 'image_url')
  final String imageUrl;
  @override
  @JsonKey(name: 'deeplink_url')
  final String deeplinkUrl;

  @override
  String toString() {
    return 'FertilizerProduct(id: $id, slug: $slug, name: $name, shortDesc: $shortDesc, imageUrl: $imageUrl, deeplinkUrl: $deeplinkUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FertilizerProductImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.slug, slug) || other.slug == slug) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.shortDesc, shortDesc) ||
                other.shortDesc == shortDesc) &&
            (identical(other.imageUrl, imageUrl) ||
                other.imageUrl == imageUrl) &&
            (identical(other.deeplinkUrl, deeplinkUrl) ||
                other.deeplinkUrl == deeplinkUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    slug,
    name,
    shortDesc,
    imageUrl,
    deeplinkUrl,
  );

  /// Create a copy of FertilizerProduct
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FertilizerProductImplCopyWith<_$FertilizerProductImpl> get copyWith =>
      __$$FertilizerProductImplCopyWithImpl<_$FertilizerProductImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$FertilizerProductImplToJson(this);
  }
}

abstract class _FertilizerProduct implements FertilizerProduct {
  const factory _FertilizerProduct({
    final String id,
    final String slug,
    final String name,
    @JsonKey(name: 'short_desc') final String shortDesc,
    @JsonKey(name: 'image_url') final String imageUrl,
    @JsonKey(name: 'deeplink_url') final String deeplinkUrl,
  }) = _$FertilizerProductImpl;

  factory _FertilizerProduct.fromJson(Map<String, dynamic> json) =
      _$FertilizerProductImpl.fromJson;

  @override
  String get id;
  @override
  String get slug;
  @override
  String get name;
  @override
  @JsonKey(name: 'short_desc')
  String get shortDesc;
  @override
  @JsonKey(name: 'image_url')
  String get imageUrl;
  @override
  @JsonKey(name: 'deeplink_url')
  String get deeplinkUrl;

  /// Create a copy of FertilizerProduct
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FertilizerProductImplCopyWith<_$FertilizerProductImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ContentBlock _$ContentBlockFromJson(Map<String, dynamic> json) {
  return _ContentBlock.fromJson(json);
}

/// @nodoc
mixin _$ContentBlock {
  String get type => throw _privateConstructorUsedError;
  String get text => throw _privateConstructorUsedError;
  @JsonKey(name: 'storage_key')
  String get storageKey => throw _privateConstructorUsedError;
  @JsonKey(name: 'duration_ms')
  int get durationMs => throw _privateConstructorUsedError;
  List<FertilizerProduct> get products => throw _privateConstructorUsedError;

  /// Serializes this ContentBlock to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ContentBlock
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ContentBlockCopyWith<ContentBlock> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ContentBlockCopyWith<$Res> {
  factory $ContentBlockCopyWith(
    ContentBlock value,
    $Res Function(ContentBlock) then,
  ) = _$ContentBlockCopyWithImpl<$Res, ContentBlock>;
  @useResult
  $Res call({
    String type,
    String text,
    @JsonKey(name: 'storage_key') String storageKey,
    @JsonKey(name: 'duration_ms') int durationMs,
    List<FertilizerProduct> products,
  });
}

/// @nodoc
class _$ContentBlockCopyWithImpl<$Res, $Val extends ContentBlock>
    implements $ContentBlockCopyWith<$Res> {
  _$ContentBlockCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ContentBlock
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? type = null,
    Object? text = null,
    Object? storageKey = null,
    Object? durationMs = null,
    Object? products = null,
  }) {
    return _then(
      _value.copyWith(
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            text: null == text
                ? _value.text
                : text // ignore: cast_nullable_to_non_nullable
                      as String,
            storageKey: null == storageKey
                ? _value.storageKey
                : storageKey // ignore: cast_nullable_to_non_nullable
                      as String,
            durationMs: null == durationMs
                ? _value.durationMs
                : durationMs // ignore: cast_nullable_to_non_nullable
                      as int,
            products: null == products
                ? _value.products
                : products // ignore: cast_nullable_to_non_nullable
                      as List<FertilizerProduct>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ContentBlockImplCopyWith<$Res>
    implements $ContentBlockCopyWith<$Res> {
  factory _$$ContentBlockImplCopyWith(
    _$ContentBlockImpl value,
    $Res Function(_$ContentBlockImpl) then,
  ) = __$$ContentBlockImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String type,
    String text,
    @JsonKey(name: 'storage_key') String storageKey,
    @JsonKey(name: 'duration_ms') int durationMs,
    List<FertilizerProduct> products,
  });
}

/// @nodoc
class __$$ContentBlockImplCopyWithImpl<$Res>
    extends _$ContentBlockCopyWithImpl<$Res, _$ContentBlockImpl>
    implements _$$ContentBlockImplCopyWith<$Res> {
  __$$ContentBlockImplCopyWithImpl(
    _$ContentBlockImpl _value,
    $Res Function(_$ContentBlockImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ContentBlock
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? type = null,
    Object? text = null,
    Object? storageKey = null,
    Object? durationMs = null,
    Object? products = null,
  }) {
    return _then(
      _$ContentBlockImpl(
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        text: null == text
            ? _value.text
            : text // ignore: cast_nullable_to_non_nullable
                  as String,
        storageKey: null == storageKey
            ? _value.storageKey
            : storageKey // ignore: cast_nullable_to_non_nullable
                  as String,
        durationMs: null == durationMs
            ? _value.durationMs
            : durationMs // ignore: cast_nullable_to_non_nullable
                  as int,
        products: null == products
            ? _value._products
            : products // ignore: cast_nullable_to_non_nullable
                  as List<FertilizerProduct>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ContentBlockImpl implements _ContentBlock {
  const _$ContentBlockImpl({
    required this.type,
    this.text = '',
    @JsonKey(name: 'storage_key') this.storageKey = '',
    @JsonKey(name: 'duration_ms') this.durationMs = 0,
    final List<FertilizerProduct> products = const <FertilizerProduct>[],
  }) : _products = products;

  factory _$ContentBlockImpl.fromJson(Map<String, dynamic> json) =>
      _$$ContentBlockImplFromJson(json);

  @override
  final String type;
  @override
  @JsonKey()
  final String text;
  @override
  @JsonKey(name: 'storage_key')
  final String storageKey;
  @override
  @JsonKey(name: 'duration_ms')
  final int durationMs;
  final List<FertilizerProduct> _products;
  @override
  @JsonKey()
  List<FertilizerProduct> get products {
    if (_products is EqualUnmodifiableListView) return _products;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_products);
  }

  @override
  String toString() {
    return 'ContentBlock(type: $type, text: $text, storageKey: $storageKey, durationMs: $durationMs, products: $products)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ContentBlockImpl &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.text, text) || other.text == text) &&
            (identical(other.storageKey, storageKey) ||
                other.storageKey == storageKey) &&
            (identical(other.durationMs, durationMs) ||
                other.durationMs == durationMs) &&
            const DeepCollectionEquality().equals(other._products, _products));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    type,
    text,
    storageKey,
    durationMs,
    const DeepCollectionEquality().hash(_products),
  );

  /// Create a copy of ContentBlock
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ContentBlockImplCopyWith<_$ContentBlockImpl> get copyWith =>
      __$$ContentBlockImplCopyWithImpl<_$ContentBlockImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ContentBlockImplToJson(this);
  }
}

abstract class _ContentBlock implements ContentBlock {
  const factory _ContentBlock({
    required final String type,
    final String text,
    @JsonKey(name: 'storage_key') final String storageKey,
    @JsonKey(name: 'duration_ms') final int durationMs,
    final List<FertilizerProduct> products,
  }) = _$ContentBlockImpl;

  factory _ContentBlock.fromJson(Map<String, dynamic> json) =
      _$ContentBlockImpl.fromJson;

  @override
  String get type;
  @override
  String get text;
  @override
  @JsonKey(name: 'storage_key')
  String get storageKey;
  @override
  @JsonKey(name: 'duration_ms')
  int get durationMs;
  @override
  List<FertilizerProduct> get products;

  /// Create a copy of ContentBlock
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ContentBlockImplCopyWith<_$ContentBlockImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChatMessage _$ChatMessageFromJson(Map<String, dynamic> json) {
  return _ChatMessage.fromJson(json);
}

/// @nodoc
mixin _$ChatMessage {
  String get id => throw _privateConstructorUsedError;
  MessageRole get role => throw _privateConstructorUsedError;
  MessageStatus get status => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  DateTime get createdAt => throw _privateConstructorUsedError;
  List<ContentBlock> get content => throw _privateConstructorUsedError;
  @JsonKey(includeFromJson: false, includeToJson: false)
  bool get streaming => throw _privateConstructorUsedError;
  @JsonKey(includeFromJson: false, includeToJson: false)
  String? get errorCode => throw _privateConstructorUsedError;

  /// Serializes this ChatMessage to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatMessage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatMessageCopyWith<ChatMessage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatMessageCopyWith<$Res> {
  factory $ChatMessageCopyWith(
    ChatMessage value,
    $Res Function(ChatMessage) then,
  ) = _$ChatMessageCopyWithImpl<$Res, ChatMessage>;
  @useResult
  $Res call({
    String id,
    MessageRole role,
    MessageStatus status,
    @JsonKey(name: 'created_at') DateTime createdAt,
    List<ContentBlock> content,
    @JsonKey(includeFromJson: false, includeToJson: false) bool streaming,
    @JsonKey(includeFromJson: false, includeToJson: false) String? errorCode,
  });
}

/// @nodoc
class _$ChatMessageCopyWithImpl<$Res, $Val extends ChatMessage>
    implements $ChatMessageCopyWith<$Res> {
  _$ChatMessageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatMessage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? role = null,
    Object? status = null,
    Object? createdAt = null,
    Object? content = null,
    Object? streaming = null,
    Object? errorCode = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            role: null == role
                ? _value.role
                : role // ignore: cast_nullable_to_non_nullable
                      as MessageRole,
            status: null == status
                ? _value.status
                : status // ignore: cast_nullable_to_non_nullable
                      as MessageStatus,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as DateTime,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as List<ContentBlock>,
            streaming: null == streaming
                ? _value.streaming
                : streaming // ignore: cast_nullable_to_non_nullable
                      as bool,
            errorCode: freezed == errorCode
                ? _value.errorCode
                : errorCode // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatMessageImplCopyWith<$Res>
    implements $ChatMessageCopyWith<$Res> {
  factory _$$ChatMessageImplCopyWith(
    _$ChatMessageImpl value,
    $Res Function(_$ChatMessageImpl) then,
  ) = __$$ChatMessageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String id,
    MessageRole role,
    MessageStatus status,
    @JsonKey(name: 'created_at') DateTime createdAt,
    List<ContentBlock> content,
    @JsonKey(includeFromJson: false, includeToJson: false) bool streaming,
    @JsonKey(includeFromJson: false, includeToJson: false) String? errorCode,
  });
}

/// @nodoc
class __$$ChatMessageImplCopyWithImpl<$Res>
    extends _$ChatMessageCopyWithImpl<$Res, _$ChatMessageImpl>
    implements _$$ChatMessageImplCopyWith<$Res> {
  __$$ChatMessageImplCopyWithImpl(
    _$ChatMessageImpl _value,
    $Res Function(_$ChatMessageImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatMessage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? role = null,
    Object? status = null,
    Object? createdAt = null,
    Object? content = null,
    Object? streaming = null,
    Object? errorCode = freezed,
  }) {
    return _then(
      _$ChatMessageImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        role: null == role
            ? _value.role
            : role // ignore: cast_nullable_to_non_nullable
                  as MessageRole,
        status: null == status
            ? _value.status
            : status // ignore: cast_nullable_to_non_nullable
                  as MessageStatus,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as DateTime,
        content: null == content
            ? _value._content
            : content // ignore: cast_nullable_to_non_nullable
                  as List<ContentBlock>,
        streaming: null == streaming
            ? _value.streaming
            : streaming // ignore: cast_nullable_to_non_nullable
                  as bool,
        errorCode: freezed == errorCode
            ? _value.errorCode
            : errorCode // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatMessageImpl implements _ChatMessage {
  const _$ChatMessageImpl({
    required this.id,
    required this.role,
    required this.status,
    @JsonKey(name: 'created_at') required this.createdAt,
    final List<ContentBlock> content = const <ContentBlock>[],
    @JsonKey(includeFromJson: false, includeToJson: false)
    this.streaming = false,
    @JsonKey(includeFromJson: false, includeToJson: false) this.errorCode,
  }) : _content = content;

  factory _$ChatMessageImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatMessageImplFromJson(json);

  @override
  final String id;
  @override
  final MessageRole role;
  @override
  final MessageStatus status;
  @override
  @JsonKey(name: 'created_at')
  final DateTime createdAt;
  final List<ContentBlock> _content;
  @override
  @JsonKey()
  List<ContentBlock> get content {
    if (_content is EqualUnmodifiableListView) return _content;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_content);
  }

  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  final bool streaming;
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  final String? errorCode;

  @override
  String toString() {
    return 'ChatMessage(id: $id, role: $role, status: $status, createdAt: $createdAt, content: $content, streaming: $streaming, errorCode: $errorCode)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatMessageImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.role, role) || other.role == role) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            const DeepCollectionEquality().equals(other._content, _content) &&
            (identical(other.streaming, streaming) ||
                other.streaming == streaming) &&
            (identical(other.errorCode, errorCode) ||
                other.errorCode == errorCode));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    role,
    status,
    createdAt,
    const DeepCollectionEquality().hash(_content),
    streaming,
    errorCode,
  );

  /// Create a copy of ChatMessage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatMessageImplCopyWith<_$ChatMessageImpl> get copyWith =>
      __$$ChatMessageImplCopyWithImpl<_$ChatMessageImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatMessageImplToJson(this);
  }
}

abstract class _ChatMessage implements ChatMessage {
  const factory _ChatMessage({
    required final String id,
    required final MessageRole role,
    required final MessageStatus status,
    @JsonKey(name: 'created_at') required final DateTime createdAt,
    final List<ContentBlock> content,
    @JsonKey(includeFromJson: false, includeToJson: false) final bool streaming,
    @JsonKey(includeFromJson: false, includeToJson: false)
    final String? errorCode,
  }) = _$ChatMessageImpl;

  factory _ChatMessage.fromJson(Map<String, dynamic> json) =
      _$ChatMessageImpl.fromJson;

  @override
  String get id;
  @override
  MessageRole get role;
  @override
  MessageStatus get status;
  @override
  @JsonKey(name: 'created_at')
  DateTime get createdAt;
  @override
  List<ContentBlock> get content;
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  bool get streaming;
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  String? get errorCode;

  /// Create a copy of ChatMessage
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatMessageImplCopyWith<_$ChatMessageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ConversationPage _$ConversationPageFromJson(Map<String, dynamic> json) {
  return _ConversationPage.fromJson(json);
}

/// @nodoc
mixin _$ConversationPage {
  List<ChatMessage> get messages => throw _privateConstructorUsedError;
  @JsonKey(name: 'next_cursor')
  String? get nextCursor => throw _privateConstructorUsedError;

  /// Serializes this ConversationPage to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ConversationPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ConversationPageCopyWith<ConversationPage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ConversationPageCopyWith<$Res> {
  factory $ConversationPageCopyWith(
    ConversationPage value,
    $Res Function(ConversationPage) then,
  ) = _$ConversationPageCopyWithImpl<$Res, ConversationPage>;
  @useResult
  $Res call({
    List<ChatMessage> messages,
    @JsonKey(name: 'next_cursor') String? nextCursor,
  });
}

/// @nodoc
class _$ConversationPageCopyWithImpl<$Res, $Val extends ConversationPage>
    implements $ConversationPageCopyWith<$Res> {
  _$ConversationPageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ConversationPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? messages = null, Object? nextCursor = freezed}) {
    return _then(
      _value.copyWith(
            messages: null == messages
                ? _value.messages
                : messages // ignore: cast_nullable_to_non_nullable
                      as List<ChatMessage>,
            nextCursor: freezed == nextCursor
                ? _value.nextCursor
                : nextCursor // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ConversationPageImplCopyWith<$Res>
    implements $ConversationPageCopyWith<$Res> {
  factory _$$ConversationPageImplCopyWith(
    _$ConversationPageImpl value,
    $Res Function(_$ConversationPageImpl) then,
  ) = __$$ConversationPageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<ChatMessage> messages,
    @JsonKey(name: 'next_cursor') String? nextCursor,
  });
}

/// @nodoc
class __$$ConversationPageImplCopyWithImpl<$Res>
    extends _$ConversationPageCopyWithImpl<$Res, _$ConversationPageImpl>
    implements _$$ConversationPageImplCopyWith<$Res> {
  __$$ConversationPageImplCopyWithImpl(
    _$ConversationPageImpl _value,
    $Res Function(_$ConversationPageImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ConversationPage
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? messages = null, Object? nextCursor = freezed}) {
    return _then(
      _$ConversationPageImpl(
        messages: null == messages
            ? _value._messages
            : messages // ignore: cast_nullable_to_non_nullable
                  as List<ChatMessage>,
        nextCursor: freezed == nextCursor
            ? _value.nextCursor
            : nextCursor // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ConversationPageImpl implements _ConversationPage {
  const _$ConversationPageImpl({
    final List<ChatMessage> messages = const <ChatMessage>[],
    @JsonKey(name: 'next_cursor') this.nextCursor,
  }) : _messages = messages;

  factory _$ConversationPageImpl.fromJson(Map<String, dynamic> json) =>
      _$$ConversationPageImplFromJson(json);

  final List<ChatMessage> _messages;
  @override
  @JsonKey()
  List<ChatMessage> get messages {
    if (_messages is EqualUnmodifiableListView) return _messages;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_messages);
  }

  @override
  @JsonKey(name: 'next_cursor')
  final String? nextCursor;

  @override
  String toString() {
    return 'ConversationPage(messages: $messages, nextCursor: $nextCursor)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ConversationPageImpl &&
            const DeepCollectionEquality().equals(other._messages, _messages) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_messages),
    nextCursor,
  );

  /// Create a copy of ConversationPage
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ConversationPageImplCopyWith<_$ConversationPageImpl> get copyWith =>
      __$$ConversationPageImplCopyWithImpl<_$ConversationPageImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ConversationPageImplToJson(this);
  }
}

abstract class _ConversationPage implements ConversationPage {
  const factory _ConversationPage({
    final List<ChatMessage> messages,
    @JsonKey(name: 'next_cursor') final String? nextCursor,
  }) = _$ConversationPageImpl;

  factory _ConversationPage.fromJson(Map<String, dynamic> json) =
      _$ConversationPageImpl.fromJson;

  @override
  List<ChatMessage> get messages;
  @override
  @JsonKey(name: 'next_cursor')
  String? get nextCursor;

  /// Create a copy of ConversationPage
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ConversationPageImplCopyWith<_$ConversationPageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
