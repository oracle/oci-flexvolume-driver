// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// Code generated. DO NOT EDIT.

// File Storage Service API
//
// APIs for OCI file storage service.
//

package ffsw

import (
    "github.com/oracle/oci-go-sdk/common"
)


    
 // ExportSetSummary Summary information for an ExportSet.
type ExportSetSummary struct {
    
 // The OCID of the compartment that contains the export set.
    CompartmentId *string `mandatory:"true" json:"compartmentId"`
    
 // A user-friendly name. Does not have to be unique, and it is changeable.
 // Avoid entering confidential information.
 // Example: `My export set`
    DisplayName *string `mandatory:"true" json:"displayName"`
    
 // The OCID of the export set.
    Id *string `mandatory:"true" json:"id"`
    
 // The current state of the export set.
    LifecycleState ExportSetSummaryLifecycleStateEnum `mandatory:"true" json:"lifecycleState"`
    
 // The date and time the export set was created, in the format defined by RFC3339.
 // Example: `2016-08-25T21:10:29.600Z`
    TimeCreated *common.SDKTime `mandatory:"true" json:"timeCreated"`
    
 // The OCID of the VCN the export set is in.
    VcnId *string `mandatory:"true" json:"vcnId"`
    
 // The Availability Domain the export set is in. May be unset.
 // Example: `Uocm:PHX-AD-1`
    AvailabilityDomain *string `mandatory:"false" json:"availabilityDomain"`
}

func (m ExportSetSummary) String() string {
    return common.PointerString(m)
}


// ExportSetSummaryLifecycleStateEnum Enum with underlying type: string
type ExportSetSummaryLifecycleStateEnum string

// Set of constants representing the allowable values for ExportSetSummaryLifecycleState
const (
    ExportSetSummaryLifecycleStateCreating ExportSetSummaryLifecycleStateEnum = "CREATING"
    ExportSetSummaryLifecycleStateActive ExportSetSummaryLifecycleStateEnum = "ACTIVE"
    ExportSetSummaryLifecycleStateDeleting ExportSetSummaryLifecycleStateEnum = "DELETING"
    ExportSetSummaryLifecycleStateDeleted ExportSetSummaryLifecycleStateEnum = "DELETED"
    ExportSetSummaryLifecycleStateUnknown ExportSetSummaryLifecycleStateEnum = "UNKNOWN"
)

var mappingExportSetSummaryLifecycleState = map[string]ExportSetSummaryLifecycleStateEnum { 
    "CREATING": ExportSetSummaryLifecycleStateCreating,
    "ACTIVE": ExportSetSummaryLifecycleStateActive,
    "DELETING": ExportSetSummaryLifecycleStateDeleting,
    "DELETED": ExportSetSummaryLifecycleStateDeleted,
    "UNKNOWN": ExportSetSummaryLifecycleStateUnknown,
}

// GetExportSetSummaryLifecycleStateEnumValues Enumerates the set of values for ExportSetSummaryLifecycleState
func GetExportSetSummaryLifecycleStateEnumValues() []ExportSetSummaryLifecycleStateEnum {
   values := make([]ExportSetSummaryLifecycleStateEnum, 0)
   for _, v := range mappingExportSetSummaryLifecycleState {
      if v != ExportSetSummaryLifecycleStateUnknown {
         values = append(values, v)
      }
   }
   return values
}



