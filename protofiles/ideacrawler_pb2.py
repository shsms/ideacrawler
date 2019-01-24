#/*************************************************************************
# *
# * Copyright 2018 Ideas2IT Technology Services Private Limited.
# *
# * Licensed under the Apache License, Version 2.0 (the "License");
# * you may not use this file except in compliance with the License.
# * You may obtain a copy of the License at
# *
# *     http://www.apache.org/licenses/LICENSE-2.0
# *
# * Unless required by applicable law or agreed to in writing, software
# * distributed under the License is distributed on an "AS IS" BASIS,
# * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# * See the License for the specific language governing permissions and
# * limitations under the License.
# *
# ***********************************************************************/

# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: ideacrawler.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf.internal import enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import timestamp_pb2 as google_dot_protobuf_dot_timestamp__pb2


DESCRIPTOR = _descriptor.FileDescriptor(
  name='ideacrawler.proto',
  package='protofiles',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=_b('\n\x11ideacrawler.proto\x12\nprotofiles\x1a\x1fgoogle/protobuf/timestamp.proto\"(\n\x06Status\x12\x0f\n\x07success\x18\x01 \x01(\x08\x12\r\n\x05\x65rror\x18\x02 \x01(\t\"!\n\x03KVP\x12\x0b\n\x03key\x18\x01 \x01(\t\x12\r\n\x05value\x18\x02 \x01(\t\"\x9d\x07\n\tDomainOpt\x12\x0f\n\x07seedUrl\x18\x01 \x01(\t\x12\x10\n\x08minDelay\x18\x02 \x01(\x05\x12\x10\n\x08maxDelay\x18\x03 \x01(\x05\x12\x10\n\x08noFollow\x18\x04 \x01(\x08\x12\x19\n\x11\x63\x61llbackUrlRegexp\x18\x05 \x01(\t\x12\x17\n\x0f\x66ollowUrlRegexp\x18\x06 \x01(\t\x12\x1d\n\x15maxConcurrentRequests\x18\x07 \x01(\x05\x12\x11\n\tuseragent\x18\x08 \x01(\t\x12\x10\n\x08impolite\x18\t \x01(\x08\x12\r\n\x05\x64\x65pth\x18\n \x01(\x05\x12+\n\x12\x63\x61llbackXpathMatch\x18\x0e \x03(\x0b\x32\x0f.protofiles.KVP\x12,\n\x13\x63\x61llbackXpathRegexp\x18\x0f \x03(\x0b\x32\x0f.protofiles.KVP\x12\x13\n\x0bmaxIdleTime\x18\x10 \x01(\x05\x12\x1a\n\x12\x66ollowOtherDomains\x18\x11 \x01(\x08\x12\x13\n\x0bkeepDomains\x18\x12 \x03(\t\x12\x13\n\x0b\x64ropDomains\x18\x13 \x03(\t\x12\x1a\n\x12\x64omainDropPriority\x18\x14 \x01(\x08\x12\x1a\n\x12unsafeNormalizeURL\x18\x15 \x01(\x08\x12\r\n\x05login\x18\x16 \x01(\x08\x12\x1a\n\x12loginUsingSelenium\x18\x17 \x01(\x08\x12\x10\n\x08loginUrl\x18\x18 \x01(\t\x12%\n\x0cloginPayload\x18\x19 \x03(\x0b\x32\x0f.protofiles.KVP\x12\x18\n\x10loginParseFields\x18\x1a \x01(\x08\x12(\n\x0floginParseXpath\x18\x1b \x03(\x0b\x32\x0f.protofiles.KVP\x12*\n\x11loginSuccessCheck\x18\x1c \x01(\x0b\x32\x0f.protofiles.KVP\x12\x1f\n\x17\x63heckLoginAfterEachPage\x18\x1d \x01(\x08\x12\x0f\n\x07loginJS\x18\x1e \x01(\t\x12\x0e\n\x06\x63hrome\x18\x1f \x01(\x08\x12\x14\n\x0c\x63hromeBinary\x18  \x01(\t\x12\x13\n\x0b\x64omLoadTime\x18! \x01(\x05\x12\x14\n\x0cnetworkIface\x18\" \x01(\t\x12\x1a\n\x12\x63\x61ncelOnDisconnect\x18# \x01(\x08\x12\x14\n\x0c\x63heckContent\x18$ \x01(\x08\x12\x10\n\x08prefetch\x18% \x01(\x08\x12 \n\x18\x63\x61llbackAnchorTextRegexp\x18\' \x01(\t\x12\x17\n\x0f\x63\x61llbackSeedUrl\x18( \x01(\x08\"\x97\x01\n\x0cSubscription\x12\x0f\n\x07subcode\x18\x01 \x01(\t\x12\x12\n\ndomainname\x18\x02 \x01(\t\x12$\n\x07subtype\x18\x03 \x01(\x0e\x32\x13.protofiles.SubType\x12\x0e\n\x06seqnum\x18\x04 \x01(\x05\x12,\n\x08\x64\x61tetime\x18\x05 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\"\x9c\x01\n\x0bPageRequest\x12%\n\x03sub\x18\x01 \x01(\x0b\x32\x18.protofiles.Subscription\x12(\n\x07reqtype\x18\x02 \x01(\x0e\x32\x17.protofiles.PageReqType\x12\x0b\n\x03url\x18\x03 \x01(\t\x12\n\n\x02js\x18\x04 \x01(\t\x12\x12\n\nnoCallback\x18\x05 \x01(\x08\x12\x0f\n\x07metaStr\x18\x06 \x01(\t\"\xbe\x01\n\x08PageHTML\x12\x0f\n\x07success\x18\x01 \x01(\x08\x12\r\n\x05\x65rror\x18\x02 \x01(\t\x12%\n\x03sub\x18\x03 \x01(\x0b\x32\x18.protofiles.Subscription\x12\x0b\n\x03url\x18\x04 \x01(\t\x12\x16\n\x0ehttpstatuscode\x18\x05 \x01(\x05\x12\x0f\n\x07\x63ontent\x18\x06 \x01(\x0c\x12\x0f\n\x07metaStr\x18\x07 \x01(\t\x12\x10\n\x08urlDepth\x18\x08 \x01(\x05\x12\x12\n\nanchorText\x18\t \x01(\t\"9\n\x07UrlList\x12\x0b\n\x03url\x18\x01 \x03(\t\x12\x0f\n\x07metaStr\x18\x02 \x01(\t\x12\x10\n\x08urlDepth\x18\x03 \x01(\x05*#\n\x07SubType\x12\n\n\x06SEQNUM\x10\x00\x12\x0c\n\x08\x44\x41TETIME\x10\x01*<\n\x0bPageReqType\x12\x07\n\x03GET\x10\x00\x12\x08\n\x04HEAD\x10\x01\x12\r\n\tBUILTINJS\x10\x02\x12\x0b\n\x07JSCRIPT\x10\x03\x32\x94\x02\n\x0bIdeaCrawler\x12\x45\n\x12\x41\x64\x64\x44omainAndListen\x12\x15.protofiles.DomainOpt\x1a\x14.protofiles.PageHTML\"\x00\x30\x01\x12;\n\x08\x41\x64\x64Pages\x12\x17.protofiles.PageRequest\x1a\x12.protofiles.Status\"\x00(\x01\x12;\n\tCancelJob\x12\x18.protofiles.Subscription\x1a\x12.protofiles.Status\"\x00\x12\x44\n\x0fGetAnalyzedURLs\x12\x18.protofiles.Subscription\x1a\x13.protofiles.UrlList\"\x00\x30\x01\x62\x06proto3')
  ,
  dependencies=[google_dot_protobuf_dot_timestamp__pb2.DESCRIPTOR,])

_SUBTYPE = _descriptor.EnumDescriptor(
  name='SubType',
  full_name='protofiles.SubType',
  filename=None,
  file=DESCRIPTOR,
  values=[
    _descriptor.EnumValueDescriptor(
      name='SEQNUM', index=0, number=0,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='DATETIME', index=1, number=1,
      serialized_options=None,
      type=None),
  ],
  containing_type=None,
  serialized_options=None,
  serialized_start=1636,
  serialized_end=1671,
)
_sym_db.RegisterEnumDescriptor(_SUBTYPE)

SubType = enum_type_wrapper.EnumTypeWrapper(_SUBTYPE)
_PAGEREQTYPE = _descriptor.EnumDescriptor(
  name='PageReqType',
  full_name='protofiles.PageReqType',
  filename=None,
  file=DESCRIPTOR,
  values=[
    _descriptor.EnumValueDescriptor(
      name='GET', index=0, number=0,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='HEAD', index=1, number=1,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='BUILTINJS', index=2, number=2,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='JSCRIPT', index=3, number=3,
      serialized_options=None,
      type=None),
  ],
  containing_type=None,
  serialized_options=None,
  serialized_start=1673,
  serialized_end=1733,
)
_sym_db.RegisterEnumDescriptor(_PAGEREQTYPE)

PageReqType = enum_type_wrapper.EnumTypeWrapper(_PAGEREQTYPE)
SEQNUM = 0
DATETIME = 1
GET = 0
HEAD = 1
BUILTINJS = 2
JSCRIPT = 3



_STATUS = _descriptor.Descriptor(
  name='Status',
  full_name='protofiles.Status',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='success', full_name='protofiles.Status.success', index=0,
      number=1, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='error', full_name='protofiles.Status.error', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=66,
  serialized_end=106,
)


_KVP = _descriptor.Descriptor(
  name='KVP',
  full_name='protofiles.KVP',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='key', full_name='protofiles.KVP.key', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='value', full_name='protofiles.KVP.value', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=108,
  serialized_end=141,
)


_DOMAINOPT = _descriptor.Descriptor(
  name='DomainOpt',
  full_name='protofiles.DomainOpt',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='seedUrl', full_name='protofiles.DomainOpt.seedUrl', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='minDelay', full_name='protofiles.DomainOpt.minDelay', index=1,
      number=2, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='maxDelay', full_name='protofiles.DomainOpt.maxDelay', index=2,
      number=3, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='noFollow', full_name='protofiles.DomainOpt.noFollow', index=3,
      number=4, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='callbackUrlRegexp', full_name='protofiles.DomainOpt.callbackUrlRegexp', index=4,
      number=5, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='followUrlRegexp', full_name='protofiles.DomainOpt.followUrlRegexp', index=5,
      number=6, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='maxConcurrentRequests', full_name='protofiles.DomainOpt.maxConcurrentRequests', index=6,
      number=7, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='useragent', full_name='protofiles.DomainOpt.useragent', index=7,
      number=8, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='impolite', full_name='protofiles.DomainOpt.impolite', index=8,
      number=9, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='depth', full_name='protofiles.DomainOpt.depth', index=9,
      number=10, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='callbackXpathMatch', full_name='protofiles.DomainOpt.callbackXpathMatch', index=10,
      number=14, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='callbackXpathRegexp', full_name='protofiles.DomainOpt.callbackXpathRegexp', index=11,
      number=15, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='maxIdleTime', full_name='protofiles.DomainOpt.maxIdleTime', index=12,
      number=16, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='followOtherDomains', full_name='protofiles.DomainOpt.followOtherDomains', index=13,
      number=17, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='keepDomains', full_name='protofiles.DomainOpt.keepDomains', index=14,
      number=18, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='dropDomains', full_name='protofiles.DomainOpt.dropDomains', index=15,
      number=19, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='domainDropPriority', full_name='protofiles.DomainOpt.domainDropPriority', index=16,
      number=20, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='unsafeNormalizeURL', full_name='protofiles.DomainOpt.unsafeNormalizeURL', index=17,
      number=21, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='login', full_name='protofiles.DomainOpt.login', index=18,
      number=22, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginUsingSelenium', full_name='protofiles.DomainOpt.loginUsingSelenium', index=19,
      number=23, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginUrl', full_name='protofiles.DomainOpt.loginUrl', index=20,
      number=24, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginPayload', full_name='protofiles.DomainOpt.loginPayload', index=21,
      number=25, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginParseFields', full_name='protofiles.DomainOpt.loginParseFields', index=22,
      number=26, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginParseXpath', full_name='protofiles.DomainOpt.loginParseXpath', index=23,
      number=27, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginSuccessCheck', full_name='protofiles.DomainOpt.loginSuccessCheck', index=24,
      number=28, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='checkLoginAfterEachPage', full_name='protofiles.DomainOpt.checkLoginAfterEachPage', index=25,
      number=29, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='loginJS', full_name='protofiles.DomainOpt.loginJS', index=26,
      number=30, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='chrome', full_name='protofiles.DomainOpt.chrome', index=27,
      number=31, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='chromeBinary', full_name='protofiles.DomainOpt.chromeBinary', index=28,
      number=32, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='domLoadTime', full_name='protofiles.DomainOpt.domLoadTime', index=29,
      number=33, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='networkIface', full_name='protofiles.DomainOpt.networkIface', index=30,
      number=34, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='cancelOnDisconnect', full_name='protofiles.DomainOpt.cancelOnDisconnect', index=31,
      number=35, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='checkContent', full_name='protofiles.DomainOpt.checkContent', index=32,
      number=36, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='prefetch', full_name='protofiles.DomainOpt.prefetch', index=33,
      number=37, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='callbackAnchorTextRegexp', full_name='protofiles.DomainOpt.callbackAnchorTextRegexp', index=34,
      number=39, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='callbackSeedUrl', full_name='protofiles.DomainOpt.callbackSeedUrl', index=35,
      number=40, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=144,
  serialized_end=1069,
)


_SUBSCRIPTION = _descriptor.Descriptor(
  name='Subscription',
  full_name='protofiles.Subscription',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='subcode', full_name='protofiles.Subscription.subcode', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='domainname', full_name='protofiles.Subscription.domainname', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='subtype', full_name='protofiles.Subscription.subtype', index=2,
      number=3, type=14, cpp_type=8, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='seqnum', full_name='protofiles.Subscription.seqnum', index=3,
      number=4, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='datetime', full_name='protofiles.Subscription.datetime', index=4,
      number=5, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1072,
  serialized_end=1223,
)


_PAGEREQUEST = _descriptor.Descriptor(
  name='PageRequest',
  full_name='protofiles.PageRequest',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='sub', full_name='protofiles.PageRequest.sub', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='reqtype', full_name='protofiles.PageRequest.reqtype', index=1,
      number=2, type=14, cpp_type=8, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='url', full_name='protofiles.PageRequest.url', index=2,
      number=3, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='js', full_name='protofiles.PageRequest.js', index=3,
      number=4, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='noCallback', full_name='protofiles.PageRequest.noCallback', index=4,
      number=5, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='metaStr', full_name='protofiles.PageRequest.metaStr', index=5,
      number=6, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1226,
  serialized_end=1382,
)


_PAGEHTML = _descriptor.Descriptor(
  name='PageHTML',
  full_name='protofiles.PageHTML',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='success', full_name='protofiles.PageHTML.success', index=0,
      number=1, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='error', full_name='protofiles.PageHTML.error', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='sub', full_name='protofiles.PageHTML.sub', index=2,
      number=3, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='url', full_name='protofiles.PageHTML.url', index=3,
      number=4, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='httpstatuscode', full_name='protofiles.PageHTML.httpstatuscode', index=4,
      number=5, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='content', full_name='protofiles.PageHTML.content', index=5,
      number=6, type=12, cpp_type=9, label=1,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='metaStr', full_name='protofiles.PageHTML.metaStr', index=6,
      number=7, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='urlDepth', full_name='protofiles.PageHTML.urlDepth', index=7,
      number=8, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='anchorText', full_name='protofiles.PageHTML.anchorText', index=8,
      number=9, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1385,
  serialized_end=1575,
)


_URLLIST = _descriptor.Descriptor(
  name='UrlList',
  full_name='protofiles.UrlList',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='url', full_name='protofiles.UrlList.url', index=0,
      number=1, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='metaStr', full_name='protofiles.UrlList.metaStr', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='urlDepth', full_name='protofiles.UrlList.urlDepth', index=2,
      number=3, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1577,
  serialized_end=1634,
)

_DOMAINOPT.fields_by_name['callbackXpathMatch'].message_type = _KVP
_DOMAINOPT.fields_by_name['callbackXpathRegexp'].message_type = _KVP
_DOMAINOPT.fields_by_name['loginPayload'].message_type = _KVP
_DOMAINOPT.fields_by_name['loginParseXpath'].message_type = _KVP
_DOMAINOPT.fields_by_name['loginSuccessCheck'].message_type = _KVP
_SUBSCRIPTION.fields_by_name['subtype'].enum_type = _SUBTYPE
_SUBSCRIPTION.fields_by_name['datetime'].message_type = google_dot_protobuf_dot_timestamp__pb2._TIMESTAMP
_PAGEREQUEST.fields_by_name['sub'].message_type = _SUBSCRIPTION
_PAGEREQUEST.fields_by_name['reqtype'].enum_type = _PAGEREQTYPE
_PAGEHTML.fields_by_name['sub'].message_type = _SUBSCRIPTION
DESCRIPTOR.message_types_by_name['Status'] = _STATUS
DESCRIPTOR.message_types_by_name['KVP'] = _KVP
DESCRIPTOR.message_types_by_name['DomainOpt'] = _DOMAINOPT
DESCRIPTOR.message_types_by_name['Subscription'] = _SUBSCRIPTION
DESCRIPTOR.message_types_by_name['PageRequest'] = _PAGEREQUEST
DESCRIPTOR.message_types_by_name['PageHTML'] = _PAGEHTML
DESCRIPTOR.message_types_by_name['UrlList'] = _URLLIST
DESCRIPTOR.enum_types_by_name['SubType'] = _SUBTYPE
DESCRIPTOR.enum_types_by_name['PageReqType'] = _PAGEREQTYPE
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

Status = _reflection.GeneratedProtocolMessageType('Status', (_message.Message,), dict(
  DESCRIPTOR = _STATUS,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.Status)
  ))
_sym_db.RegisterMessage(Status)

KVP = _reflection.GeneratedProtocolMessageType('KVP', (_message.Message,), dict(
  DESCRIPTOR = _KVP,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.KVP)
  ))
_sym_db.RegisterMessage(KVP)

DomainOpt = _reflection.GeneratedProtocolMessageType('DomainOpt', (_message.Message,), dict(
  DESCRIPTOR = _DOMAINOPT,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.DomainOpt)
  ))
_sym_db.RegisterMessage(DomainOpt)

Subscription = _reflection.GeneratedProtocolMessageType('Subscription', (_message.Message,), dict(
  DESCRIPTOR = _SUBSCRIPTION,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.Subscription)
  ))
_sym_db.RegisterMessage(Subscription)

PageRequest = _reflection.GeneratedProtocolMessageType('PageRequest', (_message.Message,), dict(
  DESCRIPTOR = _PAGEREQUEST,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.PageRequest)
  ))
_sym_db.RegisterMessage(PageRequest)

PageHTML = _reflection.GeneratedProtocolMessageType('PageHTML', (_message.Message,), dict(
  DESCRIPTOR = _PAGEHTML,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.PageHTML)
  ))
_sym_db.RegisterMessage(PageHTML)

UrlList = _reflection.GeneratedProtocolMessageType('UrlList', (_message.Message,), dict(
  DESCRIPTOR = _URLLIST,
  __module__ = 'ideacrawler_pb2'
  # @@protoc_insertion_point(class_scope:protofiles.UrlList)
  ))
_sym_db.RegisterMessage(UrlList)



_IDEACRAWLER = _descriptor.ServiceDescriptor(
  name='IdeaCrawler',
  full_name='protofiles.IdeaCrawler',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  serialized_start=1736,
  serialized_end=2012,
  methods=[
  _descriptor.MethodDescriptor(
    name='AddDomainAndListen',
    full_name='protofiles.IdeaCrawler.AddDomainAndListen',
    index=0,
    containing_service=None,
    input_type=_DOMAINOPT,
    output_type=_PAGEHTML,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='AddPages',
    full_name='protofiles.IdeaCrawler.AddPages',
    index=1,
    containing_service=None,
    input_type=_PAGEREQUEST,
    output_type=_STATUS,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='CancelJob',
    full_name='protofiles.IdeaCrawler.CancelJob',
    index=2,
    containing_service=None,
    input_type=_SUBSCRIPTION,
    output_type=_STATUS,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='GetAnalyzedURLs',
    full_name='protofiles.IdeaCrawler.GetAnalyzedURLs',
    index=3,
    containing_service=None,
    input_type=_SUBSCRIPTION,
    output_type=_URLLIST,
    serialized_options=None,
  ),
])
_sym_db.RegisterServiceDescriptor(_IDEACRAWLER)

DESCRIPTOR.services_by_name['IdeaCrawler'] = _IDEACRAWLER

# @@protoc_insertion_point(module_scope)
